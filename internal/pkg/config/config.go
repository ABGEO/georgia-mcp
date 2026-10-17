// Package config loads the configuration from the environment and the local file.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"reflect"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

const (
	prefix    = "GEOMCP_"
	localFile = ".env"
)

// FieldError tells that one field of a configuration struct has an invalid value.
// A Validate method returns it, and Load adds the name of the variable.
type FieldError struct {
	Field    string
	Problem  string
	variable string
}

// Error returns the variable, or the field when the variable is not known, and the problem.
func (e *FieldError) Error() string {
	if e.variable != "" {
		return e.variable + ": " + e.Problem
	}

	return e.Field + ": " + e.Problem
}

type validator interface {
	Validate() error
}

// Load reads the configuration into T, then calls the Validate method of each
// struct in T that has one. A variable of the environment wins over the local file.
func Load[T any]() (T, error) {
	var cfg T

	environment, err := readEnvironment()
	if err != nil {
		return cfg, fmt.Errorf("configuration: %w", err)
	}

	err = env.ParseWithOptions(&cfg, env.Options{Prefix: prefix, Environment: environment})
	if err != nil {
		err = nameParseErrors(err, reflect.TypeFor[T](), environment)

		return cfg, fmt.Errorf("configuration: %w", err)
	}

	if err := validate(reflect.ValueOf(cfg), prefix); err != nil {
		return cfg, fmt.Errorf("configuration: %w", err)
	}

	return cfg, nil
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

func validate(value reflect.Value, keyPrefix string) error {
	var errs []error

	if v, ok := reflect.TypeAssert[validator](value); ok {
		if err := v.Validate(); err != nil {
			errs = append(errs, nameFieldError(err, value.Type(), keyPrefix))
		}
	}

	for field, fieldValue := range value.Fields() {
		_, isLeaf := field.Tag.Lookup("env")
		if isLeaf || field.Type.Kind() != reflect.Struct || !field.IsExported() {
			continue
		}

		if err := validate(fieldValue, keyPrefix+field.Tag.Get("envPrefix")); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func nameFieldError(err error, structType reflect.Type, keyPrefix string) error {
	var fieldErr *FieldError
	if !errors.As(err, &fieldErr) {
		return err
	}

	if field, ok := structType.FieldByName(fieldErr.Field); ok {
		if key, ok := field.Tag.Lookup("env"); ok {
			fieldErr.variable = keyPrefix + key
		}
	}

	return err
}
