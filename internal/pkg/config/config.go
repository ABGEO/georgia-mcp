// Package config loads the configuration from the environment and the local file.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

const (
	prefix    = "GEOMCP_"
	localFile = ".env"
)

// FieldError tells that the variable of one field has an invalid value.
type FieldError struct {
	Variable string
	Problem  string
}

// Error returns the variable and the problem.
func (e *FieldError) Error() string {
	return e.Variable + ": " + e.Problem
}

// Load reads the configuration into a copy of defaults: a variable that is set replaces
// its field. Then it checks the validate tag of each field. A variable of the environment
// wins over the local file. Each invalid value gives a *FieldError.
func Load[T any](defaults T) (T, error) {
	cfg := defaults

	environment, err := readEnvironment()
	if err != nil {
		return cfg, fmt.Errorf("configuration: %w", err)
	}

	err = env.ParseWithOptions(&cfg, env.Options{Prefix: prefix, Environment: environment})
	if err != nil {
		err = nameParseErrors(err, reflect.TypeFor[T](), environment)

		return cfg, fmt.Errorf("configuration: %w", err)
	}

	if err := validate(cfg); err != nil {
		return cfg, fmt.Errorf("configuration: %w", err)
	}

	return cfg, nil
}

// URL returns the address of a default. An address that does not parse gives the empty
// URL, which the validation of Load refuses.
func URL(text string) url.URL {
	address, err := url.Parse(text)
	if err != nil {
		return url.URL{}
	}

	return *address
}

func readEnvironment() (map[string]string, error) {
	environment, err := godotenv.Read(localFile)
	if errors.Is(err, fs.ErrNotExist) {
		environment = map[string]string{}
	} else if err != nil {
		return nil, fmt.Errorf("reading %s: %w", localFile, err)
	}

	maps.Copy(environment, env.ToMap(os.Environ()))

	return environment, nil
}

// The parse error of env names the Go field, not the variable. Two structs can
// hold a field with the same name and type, so the variable that is set wins.
func nameParseErrors(err error, root reflect.Type, environment map[string]string) error {
	var aggregate env.AggregateError
	if !errors.As(err, &aggregate) {
		return err
	}

	named := make([]error, 0, len(aggregate.Errors))

	for _, inner := range aggregate.Errors {
		var parseErr env.ParseError
		if !errors.As(inner, &parseErr) {
			named = append(named, inner)

			continue
		}

		variable := parseErr.Name

		walkFields(root, prefix, func(field reflect.StructField, key string) {
			if field.Name != parseErr.Name || field.Type != parseErr.Type {
				return
			}

			if _, set := environment[key]; set {
				variable = key
			}
		})

		named = append(named, fmt.Errorf("%s: %w", variable, parseErr.Err))
	}

	return errors.Join(named...)
}

func walkFields(
	structType reflect.Type,
	keyPrefix string,
	visit func(reflect.StructField, string),
) {
	for field := range structType.Fields() {
		if key, ok := field.Tag.Lookup("env"); ok {
			visit(field, keyPrefix+key)

			continue
		}

		if field.Type.Kind() == reflect.Struct {
			walkFields(field.Type, keyPrefix+field.Tag.Get("envPrefix"), visit)
		}
	}
}

func validate(cfg any) error {
	check := validator.New(validator.WithRequiredStructEnabled())
	check.RegisterCustomTypeFunc(urlText, url.URL{})

	err := check.Struct(cfg)

	var failures validator.ValidationErrors
	if !errors.As(err, &failures) {
		return err //nolint:wrapcheck // Load adds the context.
	}

	errs := make([]error, 0, len(failures))

	for _, failure := range failures {
		place := locate(reflect.TypeOf(cfg), failure.StructNamespace())
		errs = append(errs, &FieldError{Variable: place.variable, Problem: problem(failure, place)})
	}

	return errors.Join(errs...)
}

// urlText gives the validator the text of a url.URL, so a tag of a string applies to it.
func urlText(value reflect.Value) any {
	if address, ok := reflect.TypeAssert[url.URL](value); ok {
		return address.String()
	}

	return nil
}

// place is a field of the configuration: the struct that holds it, the prefix of that
// struct, and the variable of the field.
type place struct {
	parent    reflect.Type
	keyPrefix string
	variable  string
}

// locate follows a namespace of Go fields, "Config.Rates.TBC.APIKey", from the root
// type, and adds the envPrefix of each struct on the way. An index of a list,
// "Ranges[1]", names the variable of the list.
func locate(root reflect.Type, namespace string) place {
	_, path, _ := strings.Cut(namespace, ".")
	found := place{parent: root, keyPrefix: prefix}

	for name := range strings.SplitSeq(path, ".") {
		name, _, _ = strings.Cut(name, "[")

		field, ok := found.parent.FieldByName(name)
		if !ok {
			found.variable = found.keyPrefix + name

			return found
		}

		if key, ok := field.Tag.Lookup("env"); ok {
			found.variable = found.keyPrefix + key

			return found
		}

		found.keyPrefix += field.Tag.Get("envPrefix")
		found.parent = field.Type
	}

	return found
}

// problem gives the text of a failed rule. PLATFORM spec 5 lists each one.
func problem(failure validator.FieldError, at place) string {
	switch failure.Tag() {
	case "http_url":
		return "must be an http or https URL"
	case "gt":
		return "must be more than " + failure.Param()
	case "min", "gte":
		return "must be " + failure.Param() + " or more"
	case "hostname_port":
		return "must be host:port"
	case "cidr":
		return fmt.Sprint(failure.Value()) + " must be a CIDR range"
	case "oneof":
		return "must be one of: " + strings.Join(strings.Fields(failure.Param()), ", ")
	case "required_if":
		switchName, _, _ := strings.Cut(failure.Param(), " ")
		variable := at.keyPrefix + switchName

		if field, ok := at.parent.FieldByName(switchName); ok {
			variable = at.keyPrefix + field.Tag.Get("env")
		}

		return "must be set while " + variable + " is true. Set it, or set " + variable + "=false"
	default:
		return "fails the rule " + failure.Tag()
	}
}
