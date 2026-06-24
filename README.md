# tmplfuncs

`tmplfuncs` is a small helper library for Go `text/template` and `html/template` templates.

It provides reusable, pipeline-friendly template functions for JSON rendering, inline conditionals, defaults, optional formatted output, string normalization, prefixes/suffixes, and durations.

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

## Use individual functions

All helper functions are exported, so you can register them under your own names or combine them with project-specific helpers.

```go
myFuncMap := template.FuncMap{
	"customDefault": tmplfuncs.DefaultValue,
	"json":          tmplfuncs.JSONValue,
}
```

## Helpers

| Helper       | Function name in templates | Go function       | Description                                     |
| ------------ | -------------------------- | ----------------- | ----------------------------------------------- |
| `JSON`       | `json`                     | `JSONValue`       | Render a value as a JSON literal.               |
| `When`       | `when`                     | `WhenValue`       | Return one of two strings based on a condition. |
| `Default`    | `default`                  | `DefaultValue`    | Return a fallback for empty or zero values.     |
| `Optional`   | `optional`                 | `OptionalValue`   | Render formatted text only when a value is set. |
| `Trim`       | `trim`                     | `TrimValue`       | Trim surrounding whitespace.                    |
| `Upper`      | `upper`                    | `UpperValue`      | Convert text to uppercase.                      |
| `Lower`      | `lower`                    | `LowerValue`      | Convert text to lowercase.                      |
| `WithPrefix` | `withPrefix`               | `WithPrefixValue` | Prepend a prefix when missing.                  |
| `WithSuffix` | `withSuffix`               | `WithSuffixValue` | Append a suffix when missing.                   |
| `Duration`   | `duration`                 | `DurationValue`   | Render a duration as a Go duration string.      |

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

```gotemplate
{{ when .Resolved "Resolved at" "Notified at" }}
```

### Defaults

`default` supports direct and pipeline usage.

```gotemplate
{{ default "unknown" .CheckInName }}
{{ .CheckInName | default "unknown" }}
```

### Optional formatted output

`optional` renders an empty string when the value is empty or zero.

```gotemplate
{{ optional "Status URL: %s" .App.StatusURL }}
{{ .App.StatusURL | optional "\n\n*Status URL:* %s" }}
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
func All() []Helper
func FuncMap(helpers ...Helper) template.FuncMap
func FuncMapE(helpers ...Helper) (template.FuncMap, error)

func JSONValue(value any) (string, error)
func WhenValue(condition bool, trueValue, falseValue string) string
func DefaultValue(fallback, value any) any
func OptionalValue(format string, value any) string
func TrimValue(value any) string
func UpperValue(value any) string
func LowerValue(value any) string
func WithPrefixValue(prefix string, value any) string
func WithSuffixValue(suffix string, value any) string
func DurationValue(value any) (string, error)
func IsZeroValue(value any) bool
func StringValue(value any) string
```

## Compatibility

The module targets Go 1.23 or newer.

The helpers work with both `text/template` and `html/template` because both accept `template.FuncMap` values with the same underlying type.
