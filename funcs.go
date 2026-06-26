// Package tmplfuncs provides small, reusable helper functions for Go templates.
//
// The helpers are designed for use with text/template and html/template. They
// are pipeline-friendly where that makes templates easier to read.
package tmplfuncs

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"text/template"
	"time"
)

// Helper identifies a template helper that can be registered in a FuncMap.
//
// The constant name is exported for Go code. Its string value is the function
// name registered inside templates.
type Helper string

const (
	// JSON registers json, which renders a value as a JSON literal.
	JSON Helper = "json"

	// When registers when, which returns one of two values based on template truthiness.
	When Helper = "when"

	// Default registers default, which returns a fallback for empty values.
	Default Helper = "default"

	// Coalesce registers coalesce, which returns the first non-empty value.
	Coalesce Helper = "coalesce"

	// Optional registers optional, which formats text only when all values are non-empty.
	Optional Helper = "optional"

	// FormatTime registers formatTime, which formats a time value with a Go layout.
	FormatTime Helper = "formatTime"

	// Trim registers trim, which removes surrounding whitespace.
	Trim Helper = "trim"

	// Upper registers upper, which converts text to uppercase.
	Upper Helper = "upper"

	// Lower registers lower, which converts text to lowercase.
	Lower Helper = "lower"

	// WithPrefix registers withPrefix, which prepends a prefix when missing.
	WithPrefix Helper = "withPrefix"

	// WithSuffix registers withSuffix, which appends a suffix when missing.
	WithSuffix Helper = "withSuffix"

	// Duration registers duration, which renders a time.Duration or duration string.
	Duration Helper = "duration"
)

// All returns all built-in helpers in deterministic order.
func All() []Helper {
	return []Helper{
		JSON,
		When,
		Default,
		Coalesce,
		Optional,
		FormatTime,
		Trim,
		Upper,
		Lower,
		WithPrefix,
		WithSuffix,
		Duration,
	}
}

// helperDefinition describes how one helper is exposed to templates.
type helperDefinition struct {
	name string
	fn   any
}

// helperDefinitions maps public helper identifiers to their template names and implementations.
var helperDefinitions = map[Helper]helperDefinition{
	JSON:       {name: string(JSON), fn: JSONValue},
	When:       {name: string(When), fn: WhenValue},
	Default:    {name: string(Default), fn: DefaultValue},
	Coalesce:   {name: string(Coalesce), fn: CoalesceValue},
	Optional:   {name: string(Optional), fn: OptionalValue},
	FormatTime: {name: string(FormatTime), fn: FormatTimeValue},
	Trim:       {name: string(Trim), fn: TrimValue},
	Upper:      {name: string(Upper), fn: UpperValue},
	Lower:      {name: string(Lower), fn: LowerValue},
	WithPrefix: {name: string(WithPrefix), fn: WithPrefixValue},
	WithSuffix: {name: string(WithSuffix), fn: WithSuffixValue},
	Duration:   {name: string(Duration), fn: DurationValue},
}

// FuncMap returns a template.FuncMap containing the selected helpers.
//
// When called without arguments, FuncMap returns all helpers. When called with
// helpers, it returns only those helpers.
//
// FuncMap panics for unknown helper identifiers because invalid helper
// identifiers are programming errors. Use FuncMapE when helper identifiers come
// from dynamic input.
func FuncMap(helpers ...Helper) template.FuncMap {
	funcs, err := FuncMapE(helpers...)
	if err != nil {
		panic(err)
	}
	return funcs
}

// FuncMapE returns a template.FuncMap containing the selected helpers.
//
// When called without arguments, FuncMapE returns all helpers. When called with
// helpers, it returns only those helpers.
//
// FuncMapE returns an error for unknown helper identifiers.
func FuncMapE(helpers ...Helper) (template.FuncMap, error) {
	if len(helpers) == 0 {
		helpers = All()
	}

	funcs := make(template.FuncMap, len(helpers))
	for _, helper := range helpers {
		definition, ok := helperDefinitions[helper]
		if !ok {
			return nil, fmt.Errorf("unknown template helper %q", helper)
		}
		funcs[definition.name] = definition.fn
	}

	return funcs, nil
}

// JSONValue renders value as a JSON literal.
//
// This is useful when embedding dynamic strings, maps, slices, or structs in
// JSON templates.
//
//	"text": {{ .Subject | json }}
func JSONValue(value any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// WhenValue returns trueValue when condition is truthy, otherwise falseValue.
//
// Truthiness follows template-style rules: nil, false, zero numbers, empty
// strings, empty arrays, empty maps, empty slices, empty channels, and nil
// pointers/interfaces are false. Other values are true.
//
// The helper accepts any value type so it can be used with booleans, strings,
// numbers, slices, maps, and pointers.
//
//	{{ when .Resolved "Resolved at" "Notified at" }}
//	{{ when .Status "has status" "missing status" }}
func WhenValue(condition, trueValue, falseValue any) any {
	if isTruthyValue(condition) {
		return trueValue
	}
	return falseValue
}

// DefaultValue returns fallback when value is empty.
//
// Empty follows the same rules as template helpers: nil, false, zero numbers,
// empty strings, empty arrays, empty maps, empty slices, empty channels, nil
// pointers/interfaces, and other Go zero values are empty.
//
// The argument order supports both direct and pipeline usage:
//
//	{{ default "fallback" .Value }}
//	{{ .Value | default "fallback" }}
func DefaultValue(fallback, value any) any {
	if isEmptyValue(value) {
		return fallback
	}
	return value
}

// CoalesceValue returns the first non-empty value.
//
// Empty follows the same rules as DefaultValue.
//
//	{{ coalesce .Title .Name "unknown" }}
func CoalesceValue(values ...any) any {
	for _, value := range values {
		if !isEmptyValue(value) {
			return value
		}
	}
	return nil
}

// OptionalValue formats text only when all values are non-empty.
//
// Empty follows the same rules as DefaultValue. Values are converted to strings
// and trimmed before formatting, so whitespace-only values are treated as empty.
//
// The argument order supports direct and pipeline usage:
//
//	{{ optional "Status URL: %s" .App.StatusURL }}
//	{{ optional "%s: %s" .Label .Value }}
//	{{ .App.StatusURL | optional "Status URL: %s" }}
func OptionalValue(format string, values ...any) string {
	if format == "" || len(values) == 0 {
		return ""
	}

	args := make([]any, 0, len(values))
	for _, value := range values {
		if isEmptyValue(value) {
			return ""
		}

		text := strings.TrimSpace(stringValue(value))
		if text == "" {
			return ""
		}

		args = append(args, text)
	}

	return fmt.Sprintf(format, args...)
}

// FormatTimeValue formats a time value with layout.
//
// The value may be time.Time, *time.Time, or an RFC3339/RFC3339Nano string.
// The argument order supports both direct and pipeline usage:
//
//	{{ formatTime "2006-01-02 15:04:05 MST" .ExpectedBy }}
//	{{ .ExpectedBy | formatTime "2006-01-02 15:04:05 MST" }}
func FormatTimeValue(layout string, value any) (string, error) {
	t, err := timeValue(value)
	if err != nil {
		return "", err
	}
	return t.Format(layout), nil
}

// TrimValue returns value as a string with surrounding whitespace removed.
//
//	{{ .Name | trim }}
func TrimValue(value any) string {
	return strings.TrimSpace(stringValue(value))
}

// UpperValue returns value as an uppercase string.
//
//	{{ .Status | upper }}
func UpperValue(value any) string {
	return strings.ToUpper(stringValue(value))
}

// LowerValue returns value as a lowercase string.
//
//	{{ .Status | lower }}
func LowerValue(value any) string {
	return strings.ToLower(stringValue(value))
}

// WithPrefixValue returns value with prefix prepended when it is not already present.
//
// The value and prefix are trimmed before comparison. The argument order
// supports both direct and pipeline usage:
//
//	{{ withPrefix "#" .CustomData.channel }}
//	{{ .CustomData.channel | withPrefix "#" }}
func WithPrefixValue(prefix string, value any) string {
	text := strings.TrimSpace(stringValue(value))
	prefix = strings.TrimSpace(prefix)

	if text == "" || prefix == "" || strings.HasPrefix(text, prefix) {
		return text
	}

	return prefix + text
}

// WithSuffixValue returns value with suffix appended when it is not already present.
//
// The value and suffix are trimmed before comparison. The argument order
// supports both direct and pipeline usage:
//
//	{{ withSuffix "/" .Path }}
//	{{ .Path | withSuffix "/" }}
func WithSuffixValue(suffix string, value any) string {
	text := strings.TrimSpace(stringValue(value))
	suffix = strings.TrimSpace(suffix)

	if text == "" || suffix == "" || strings.HasSuffix(text, suffix) {
		return text
	}

	return text + suffix
}

// DurationValue renders a duration value as a Go duration string.
//
// The value may be time.Duration, *time.Duration, or a string accepted by
// time.ParseDuration.
//
//	{{ .AlertingDelay | duration }}
func DurationValue(value any) (string, error) {
	d, err := durationValue(value)
	if err != nil {
		return "", err
	}
	return d.String(), nil
}

// isTruthyValue reports whether value should be treated as true in template helpers.
func isTruthyValue(value any) bool {
	if value == nil {
		return false
	}

	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return false
	}

	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.String, reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return v.Len() > 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return v.Float() != 0
	case reflect.Interface, reflect.Pointer:
		return !v.IsNil() && isTruthyValue(v.Elem().Interface())
	default:
		return true
	}
}

// isEmptyValue reports whether value should be treated as empty in template helpers.
func isEmptyValue(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.String, reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return v.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	default:
		return v.IsZero()
	}
}

// stringValue converts a template value to a string.
//
// Nil values render as an empty string.
func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

// timeValue converts supported template inputs into a time.Time.
func timeValue(value any) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case *time.Time:
		if v == nil {
			return time.Time{}, fmt.Errorf("time value must not be nil")
		}
		return *v, nil
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return time.Time{}, fmt.Errorf("time value must not be empty")
		}

		t, err := time.Parse(time.RFC3339Nano, text)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse time %q: %w", v, err)
		}
		return t, nil
	default:
		return time.Time{}, fmt.Errorf("time value must be time.Time or RFC3339 string, got %T", value)
	}
}

// durationValue converts supported template inputs into a time.Duration.
func durationValue(value any) (time.Duration, error) {
	switch v := value.(type) {
	case time.Duration:
		return v, nil
	case *time.Duration:
		if v == nil {
			return 0, fmt.Errorf("duration value must not be nil")
		}
		return *v, nil
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return 0, fmt.Errorf("duration value must not be empty")
		}

		d, err := time.ParseDuration(text)
		if err != nil {
			return 0, fmt.Errorf("parse duration %q: %w", v, err)
		}
		return d, nil
	default:
		return 0, fmt.Errorf("duration value must be time.Duration or duration string, got %T", value)
	}
}
