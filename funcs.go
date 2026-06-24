// Package tmplfuncs provides small, reusable helper functions for Go templates.
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

	// When registers when, which returns one of two strings based on a boolean condition.
	When Helper = "when"

	// Default registers default, which returns a fallback for empty or zero values.
	Default Helper = "default"

	// Coalesce registers coalesce, which returns the first non-empty, non-zero value.
	Coalesce Helper = "coalesce"

	// Optional registers optional, which renders formatted text only when a value is set.
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
// helpers, it returns only those helpers. FuncMap panics for unknown helper
// identifiers because invalid helper constants are programming errors.
func FuncMap(helpers ...Helper) template.FuncMap {
	if len(helpers) == 0 {
		helpers = All()
	}

	funcs := make(template.FuncMap, len(helpers))
	for _, helper := range helpers {
		definition, ok := helperDefinitions[helper]
		if !ok {
			panic(fmt.Sprintf("unknown template helper %q", helper))
		}
		funcs[definition.name] = definition.fn
	}

	return funcs
}

// JSONValue renders any template value as a JSON literal.
func JSONValue(value any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// WhenValue returns trueValue when condition is true and falseValue otherwise.
//
// The argument order favors natural inline conditional usage:
//
//	{{ when .Resolved "Resolved at" "Notified at" }}
func WhenValue(condition bool, trueValue, falseValue string) string {
	if condition {
		return trueValue
	}
	return falseValue
}

// DefaultValue returns fallback when value is empty or the zero value.
//
// The argument order supports both direct and pipeline usage:
//
//	{{ default "fallback" .Value }}
//	{{ .Value | default "fallback" }}
func DefaultValue(fallback, value any) any {
	if IsZeroValue(value) {
		return fallback
	}
	return value
}

// CoalesceValue returns the first value that is not empty or the zero value.
func CoalesceValue(values ...any) any {
	for _, value := range values {
		if !IsZeroValue(value) {
			return value
		}
	}
	return nil
}

// OptionalValue returns formatted text when value is not empty.
//
// The argument order supports both direct and pipeline usage:
//
//	{{ optional "Status URL: %s" .App.StatusURL }}
//	{{ .App.StatusURL | optional "Status URL: %s" }}
func OptionalValue(format string, value any) string {
	if IsZeroValue(value) {
		return ""
	}

	text := strings.TrimSpace(StringValue(value))
	if text == "" {
		return ""
	}

	return fmt.Sprintf(format, text)
}

// FormatTimeValue formats a time value with layout.
//
// The value may be a time.Time, *time.Time, or an RFC3339/RFC3339Nano string.
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
func TrimValue(value any) string {
	return strings.TrimSpace(StringValue(value))
}

// UpperValue returns value as an uppercase string.
func UpperValue(value any) string {
	return strings.ToUpper(StringValue(value))
}

// LowerValue returns value as a lowercase string.
func LowerValue(value any) string {
	return strings.ToLower(StringValue(value))
}

// WithPrefixValue returns value with prefix prepended when it is not already present.
//
// The argument order supports both direct and pipeline usage:
//
//	{{ withPrefix "#" .CustomData.channel }}
//	{{ .CustomData.channel | withPrefix "#" }}
func WithPrefixValue(prefix string, value any) string {
	if value == nil {
		return ""
	}

	text := strings.TrimSpace(fmt.Sprint(value))
	prefix = strings.TrimSpace(prefix)

	if text == "" || prefix == "" || strings.HasPrefix(text, prefix) {
		return text
	}

	return prefix + text
}

// WithSuffixValue returns value with suffix appended when it is not already present.
//
// The argument order supports both direct and pipeline usage:
//
//	{{ withSuffix "/" .CustomData.path }}
//	{{ .CustomData.path | withSuffix "/" }}
func WithSuffixValue(suffix string, value any) string {
	if value == nil {
		return ""
	}

	text := strings.TrimSpace(fmt.Sprint(value))
	suffix = strings.TrimSpace(suffix)

	if text == "" || suffix == "" || strings.HasSuffix(text, suffix) {
		return text
	}

	return text + suffix
}

// DurationValue renders a duration value as a Go duration string.
//
// The value may be a time.Duration, *time.Duration, or a string accepted by
// time.ParseDuration.
func DurationValue(value any) (string, error) {
	d, err := durationValue(value)
	if err != nil {
		return "", err
	}
	return d.String(), nil
}

// IsZeroValue reports whether value is nil, invalid, or the zero value for its type.
func IsZeroValue(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if v.IsNil() {
			return true
		}
	}

	return v.IsZero()
}

// StringValue converts a template value to a string.
func StringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

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
		t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(v))
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
		d, err := time.ParseDuration(strings.TrimSpace(v))
		if err != nil {
			return 0, fmt.Errorf("parse duration %q: %w", v, err)
		}
		return d, nil
	default:
		return 0, fmt.Errorf("duration value must be time.Duration or duration string, got %T", value)
	}
}
