# tmplfuncs

`tmplfuncs` is a small helper library for Go `text/template` and `html/template` templates.

It provides reusable, pipeline-friendly template functions for JSON rendering, inline conditionals, defaults, coalescing, optional formatted output, time formatting, string normalization, prefixes/suffixes, and durations.

It intentionally avoids locale-dependent helpers such as `ago`, because human-readable relative time usually needs translation and project-specific wording.

## Install

```sh
go get github.com/containeroo/tmplfuncs
```

## Register all helpers

```go
package main

import (
	"text/template"

	"github.com/containeroo/tmplfuncs"
)

func main() {
	tmpl, err := template.New("message").
		Funcs(tmplfuncs.FuncMap()).
		Parse(`{{ .Title | json }}`)
	_ = tmpl
	_ = err
}
```

## Register selected helpers

`FuncMap` accepts optional helper identifiers. When no helpers are passed, all helpers are registered. When helpers are passed, only those helpers are registered.

```go
package main

import (
	"text/template"

	"github.com/containeroo/tmplfuncs"
)

func main() {
	tmpl, err := template.New("message").
		Funcs(tmplfuncs.FuncMap(
			tmplfuncs.JSON,
			tmplfuncs.When,
			tmplfuncs.Default,
			tmplfuncs.Optional,
		)).
		Parse(`{{ when .Resolved "up" "down" }}{{ .StatusURL | optional " %s" }}`)
	_ = tmpl
	_ = err
}
```

`FuncMap` panics when an unknown helper identifier is passed. Use `FuncMapE` when you want an error instead:

```go
funcs, err := tmplfuncs.FuncMapE(tmplfuncs.JSON, tmplfuncs.Optional)
```

## Use individual helper functions

The helper functions registered by `FuncMap` are exported, so you can register them under your own names or combine them with project-specific helpers.

```go
myFuncMap := template.FuncMap{
	"customDefault": tmplfuncs.DefaultValue,
	"json":          tmplfuncs.JSONValue,
}
```

Internal support helpers are intentionally not part of the public API.

## Helpers

| Helper       | Function name in templates | Go function       | Description                                    |
| ------------ | -------------------------- | ----------------- | ---------------------------------------------- |
| `JSON`       | `json`                     | `JSONValue`       | Render a value as a JSON literal.              |
| `When`       | `when`                     | `WhenValue`       | Return one of two values based on truthiness.  |
| `Default`    | `default`                  | `DefaultValue`    | Return a fallback for empty values.            |
| `Coalesce`   | `coalesce`                 | `CoalesceValue`   | Return the first non-empty value.              |
| `Optional`   | `optional`                 | `OptionalValue`   | Render formatted text when all values are set. |
| `FormatTime` | `formatTime`               | `FormatTimeValue` | Format a time value with a Go layout.          |
| `Trim`       | `trim`                     | `TrimValue`       | Trim surrounding whitespace.                   |
| `Upper`      | `upper`                    | `UpperValue`      | Convert text to uppercase.                     |
| `Lower`      | `lower`                    | `LowerValue`      | Convert text to lowercase.                     |
| `WithPrefix` | `withPrefix`               | `WithPrefixValue` | Prepend a prefix when missing.                 |
| `WithSuffix` | `withSuffix`               | `WithSuffixValue` | Append a suffix when missing.                  |
| `Duration`   | `duration`                 | `DurationValue`   | Render a duration as a Go duration string.     |

## Template examples

### JSON

Use `json` when rendering dynamic values into JSON templates.

```gotemplate
{
  "text": {{ .Text | json }},
  "status": {{ .Status | json }}
}
```

### Inline conditionals

`when` returns the second argument when the condition is truthy, otherwise the third argument.

```gotemplate
{{ when .Resolved "Resolved at" "Notified at" }}
{{ when .Status "has status" "missing status" }}
```

### Defaults

`default` supports direct and pipeline usage.

```gotemplate
{{ default "unknown" .CheckInName }}
{{ .CheckInName | default "unknown" }}
```

Empty means nil, false, zero numbers, empty strings, empty arrays, empty maps, empty slices, empty channels, nil pointers/interfaces, and other Go zero values.

### Coalescing

`coalesce` returns the first non-empty value.

```gotemplate
{{ coalesce .Title .Name "unknown" }}
```

### Optional formatted output

`optional` renders an empty string when the format is empty, no values are provided, or any value is empty.

```gotemplate
{{ optional "Status URL: %s" .App.StatusURL }}
{{ optional "%s: %s" .Label .Value }}
{{ .App.StatusURL | optional "\n\n*Status URL:* %s" }}
```

Values are converted to strings and trimmed before formatting.

### Time formatting

`formatTime` accepts `time.Time`, `*time.Time`, or an RFC3339/RFC3339Nano string.

```gotemplate
{{ .ExpectedBy | formatTime "2006-01-02 15:04:05 MST" }}
```

### Prefix and suffix helpers

```gotemplate
{{ .CustomData.channel | default "alertmanager" | withPrefix "#" }}
{{ .Path | withSuffix "/" }}
```

### Duration

`duration` accepts `time.Duration`, `*time.Duration`, or a duration string accepted by `time.ParseDuration`.

```gotemplate
{{ .AlertingDelay | duration }}
```

## API

```go
type Helper string

const (
	JSON       Helper = "json"
	When       Helper = "when"
	Default    Helper = "default"
	Coalesce   Helper = "coalesce"
	Optional   Helper = "optional"
	FormatTime Helper = "formatTime"
	Trim       Helper = "trim"
	Upper      Helper = "upper"
	Lower      Helper = "lower"
	WithPrefix Helper = "withPrefix"
	WithSuffix Helper = "withSuffix"
	Duration   Helper = "duration"
)

func All() []Helper
func FuncMap(helpers ...Helper) template.FuncMap
func FuncMapE(helpers ...Helper) (template.FuncMap, error)

func JSONValue(value any) (string, error)
func WhenValue(condition, trueValue, falseValue any) any
func DefaultValue(fallback, value any) any
func CoalesceValue(values ...any) any
func OptionalValue(format string, values ...any) string
func FormatTimeValue(layout string, value any) (string, error)
func TrimValue(value any) string
func UpperValue(value any) string
func LowerValue(value any) string
func WithPrefixValue(prefix string, value any) string
func WithSuffixValue(suffix string, value any) string
func DurationValue(value any) (string, error)
```

## Compatibility

The module targets Go 1.23 or newer.

The helpers work with both `text/template` and `html/template` because both accept `template.FuncMap` values with the same underlying type.

## License

This project is licensed under the Apache 2.0 License. See the [LICENSE](LICENSE) file for details.
