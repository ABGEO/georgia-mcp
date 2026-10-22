// Package vendors builds the vendors of a context: each vendor that the operator turned
// on, in the order that the context adds them.
package vendors

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
)

// Entry is one vendor of a context, with the switch of the operator.
type Entry[T io.Closer] struct {
	// Name names the vendor in the log: "Lukoil".
	Name    string
	Enabled bool
	// Variable is the switch of the vendor: "GEOMCP_FUEL_LUKOIL_ENABLED".
	Variable string
	// Build returns the source of the vendor. The registry calls it only when the vendor
	// is on.
	Build func() (T, error)
}

// Registry holds the vendors of one context.
type Registry[T io.Closer] struct {
	logger  *slog.Logger
	entries []Entry[T]
}

// New returns an empty registry that logs to logger.
func New[T io.Closer](logger *slog.Logger) *Registry[T] {
	return &Registry[T]{logger: logger}
}

// Add adds a vendor, whether it is on or off.
func (r *Registry[T]) Add(entry Entry[T]) {
	r.entries = append(r.entries, entry)
}

// Build returns the source of each vendor that is on, in the order of Add. It logs each
// vendor that is off. When a vendor cannot start, it closes each source that it built and
// returns the error.
func (r *Registry[T]) Build() ([]T, error) {
	var sources []T

	for _, entry := range r.entries {
		if !entry.Enabled {
			r.logger.Info(entry.Name+" is off", slog.String("variable", entry.Variable))

			continue
		}

		source, err := entry.Build()
		if err != nil {
			return nil, errors.Join(
				fmt.Errorf("building %s: %w", entry.Name, err),
				CloseAll(sources),
			)
		}

		sources = append(sources, source)
	}

	return sources, nil
}

// CloseAll closes each source and joins the errors.
func CloseAll[T io.Closer](sources []T) error {
	errs := make([]error, 0, len(sources))

	for _, source := range sources {
		errs = append(errs, source.Close())
	}

	return errors.Join(errs...)
}
