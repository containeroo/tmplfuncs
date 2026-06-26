package tmplfuncs

import (
	"errors"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuncMap(t *testing.T) {
	t.Parallel()

	t.Run("returns all helpers by default", func(t *testing.T) {
		t.Parallel()

		funcs := FuncMap()

		assert.Contains(t, funcs, "json")
		assert.Contains(t, funcs, "when")
		assert.Contains(t, funcs, "default")
		assert.Contains(t, funcs, "coalesce")
		assert.Contains(t, funcs, "optional")
		assert.Contains(t, funcs, "formatTime")
		assert.Contains(t, funcs, "trim")
		assert.Contains(t, funcs, "upper")
		assert.Contains(t, funcs, "lower")
		assert.Contains(t, funcs, "withPrefix")
		assert.Contains(t, funcs, "withSuffix")
		assert.Contains(t, funcs, "duration")
	})

	t.Run("returns selected helpers", func(t *testing.T) {
		t.Parallel()

		funcs := FuncMap(JSON, Optional, Coalesce, FormatTime)

		assert.Len(t, funcs, 4)
		assert.Contains(t, funcs, "json")
		assert.Contains(t, funcs, "optional")
		assert.Contains(t, funcs, "coalesce")
		assert.Contains(t, funcs, "formatTime")
		assert.NotContains(t, funcs, "when")
		assert.NotContains(t, funcs, "default")
	})

	t.Run("panics for unknown helpers", func(t *testing.T) {
		t.Parallel()

		require.PanicsWithValue(t, `unknown template helper "missing"`, func() {
			_ = FuncMap(Helper("missing"))
		})
	})
}

func TestFuncMapE(t *testing.T) {
	t.Parallel()

	t.Run("returns selected helpers", func(t *testing.T) {
		t.Parallel()

		funcs, err := FuncMapE(JSON, Optional)

		require.NoError(t, err)
		assert.Len(t, funcs, 2)
		assert.Contains(t, funcs, "json")
		assert.Contains(t, funcs, "optional")
	})

	t.Run("returns error for unknown helpers", func(t *testing.T) {
		t.Parallel()

		funcs, err := FuncMapE(Helper("missing"))

		require.Error(t, err)
		assert.Nil(t, funcs)
		assert.EqualError(t, err, `unknown template helper "missing"`)
	})
}

func TestAll(t *testing.T) {
	t.Parallel()

	helpers := All()

	require.Len(t, helpers, 12)
	assert.Equal(t, JSON, helpers[0])
	assert.Equal(t, When, helpers[1])
	assert.Equal(t, Default, helpers[2])
	assert.Equal(t, Coalesce, helpers[3])
	assert.Equal(t, Optional, helpers[4])
	assert.Equal(t, FormatTime, helpers[5])
	assert.Equal(t, Trim, helpers[6])
	assert.Equal(t, Upper, helpers[7])
	assert.Equal(t, Lower, helpers[8])
	assert.Equal(t, WithPrefix, helpers[9])
	assert.Equal(t, WithSuffix, helpers[10])
	assert.Equal(t, Duration, helpers[11])

	helpers[0] = Helper("changed")
	assert.Equal(t, JSON, All()[0])
}

func TestTemplatePipelines(t *testing.T) {
	t.Parallel()

	tmpl, err := template.New("message").Funcs(FuncMap()).Parse(
		`{{ .Channel | default "alertmanager" | withPrefix "#" }} {{ when .Resolved "up" "down" }} {{ coalesce .Owner .Team "unknown" }}{{ .StatusURL | optional " %s" }} {{ .ExpectedBy | formatTime "2006-01-02 15:04:05 MST" }}`,
	)
	require.NoError(t, err)

	var out strings.Builder
	err = tmpl.Execute(&out, map[string]any{
		"Channel":    "alerts",
		"Resolved":   true,
		"Owner":      "",
		"Team":       "platform",
		"StatusURL":  "https://overdue.example.test/status",
		"ExpectedBy": time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	assert.Equal(t, "#alerts up platform https://overdue.example.test/status 2026-06-24 12:00:00 UTC", out.String())
}

func TestJSONValue(t *testing.T) {
	t.Parallel()

	t.Run("renders json literal", func(t *testing.T) {
		t.Parallel()

		got, err := JSONValue(map[string]any{"name": "prometheus", "count": float64(2)})

		require.NoError(t, err)
		assert.JSONEq(t, `{"count":2,"name":"prometheus"}`, got)
	})

	t.Run("returns marshal errors", func(t *testing.T) {
		t.Parallel()

		_, err := JSONValue(func() {})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported type")
	})
}

func TestWhenValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "yes", WhenValue(true, "yes", "no"))
	assert.Equal(t, "no", WhenValue(false, "yes", "no"))
	assert.Equal(t, "yes", WhenValue("value", "yes", "no"))
	assert.Equal(t, "no", WhenValue("", "yes", "no"))
	assert.Equal(t, "yes", WhenValue(1, "yes", "no"))
	assert.Equal(t, "no", WhenValue(0, "yes", "no"))
	assert.Equal(t, "yes", WhenValue([]string{"value"}, "yes", "no"))
	assert.Equal(t, "no", WhenValue([]string{}, "yes", "no"))
	assert.Equal(t, 1, WhenValue(true, 1, 2))
}

func TestDefaultValue(t *testing.T) {
	t.Parallel()

	t.Run("returns fallback for empty values", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "fallback", DefaultValue("fallback", ""))
		assert.Equal(t, "fallback", DefaultValue("fallback", 0))
		assert.Equal(t, "fallback", DefaultValue("fallback", false))
		assert.Equal(t, "fallback", DefaultValue("fallback", []string{}))
		assert.Equal(t, "fallback", DefaultValue("fallback", map[string]string{}))
		assert.Equal(t, "fallback", DefaultValue("fallback", time.Time{}))
	})

	t.Run("returns non-empty values", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "value", DefaultValue("fallback", "value"))
		assert.Equal(t, 42, DefaultValue("fallback", 42))
		assert.Equal(t, true, DefaultValue("fallback", true))
		assert.Equal(t, []string{"value"}, DefaultValue("fallback", []string{"value"}))
	})
}

func TestCoalesceValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "platform", CoalesceValue("", 0, []string{}, map[string]string{}, "platform", "fallback"))
	assert.Equal(t, "fallback", CoalesceValue(nil, false, "fallback"))
	assert.Nil(t, CoalesceValue("", 0, false, nil, []string{}, map[string]string{}))
}

func TestOptionalValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Status URL: https://example.test/status", OptionalValue("Status URL: %s", " https://example.test/status "))
	assert.Equal(t, "Label: value", OptionalValue("%s: %s", " Label ", " value "))
	assert.Empty(t, OptionalValue("Status URL: %s", ""))
	assert.Empty(t, OptionalValue("Status URL: %s", "   "))
	assert.Empty(t, OptionalValue("Status URL: %s", nil))
	assert.Empty(t, OptionalValue("%s: %s", "Label", ""))
	assert.Empty(t, OptionalValue(""))
	assert.Empty(t, OptionalValue("Status URL: %s"))
}

func TestFormatTimeValue(t *testing.T) {
	t.Parallel()

	t.Run("formats time value", func(t *testing.T) {
		t.Parallel()

		value := time.Date(2026, 6, 24, 12, 0, 0, 123, time.UTC)

		got, err := FormatTimeValue("2006-01-02 15:04:05 MST", value)

		require.NoError(t, err)
		assert.Equal(t, "2026-06-24 12:00:00 UTC", got)
	})

	t.Run("formats time pointer", func(t *testing.T) {
		t.Parallel()

		value := time.Date(2026, 6, 24, 12, 0, 0, 123, time.UTC)

		got, err := FormatTimeValue("2006-01-02 15:04:05 MST", &value)

		require.NoError(t, err)
		assert.Equal(t, "2026-06-24 12:00:00 UTC", got)
	})

	t.Run("formats rfc3339 string", func(t *testing.T) {
		t.Parallel()

		value := time.Date(2026, 6, 24, 12, 0, 0, 123, time.UTC)

		got, err := FormatTimeValue("2006-01-02 15:04:05 MST", value.Format(time.RFC3339Nano))

		require.NoError(t, err)
		assert.Equal(t, "2026-06-24 12:00:00 UTC", got)
	})

	t.Run("rejects empty time string", func(t *testing.T) {
		t.Parallel()

		_, err := FormatTimeValue(time.RFC3339, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "time value must not be empty")
	})

	t.Run("rejects invalid time string", func(t *testing.T) {
		t.Parallel()

		_, err := FormatTimeValue(time.RFC3339, "bad")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse time")
	})

	t.Run("rejects nil time pointer", func(t *testing.T) {
		t.Parallel()

		var value *time.Time

		_, err := FormatTimeValue(time.RFC3339, value)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "time value must not be nil")
	})

	t.Run("rejects unsupported value", func(t *testing.T) {
		t.Parallel()

		_, err := FormatTimeValue(time.RFC3339, 42)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "got int")
	})
}

func TestStringHelpers(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "value", TrimValue(" value\n"))
	assert.Equal(t, "42", TrimValue(42))
	assert.Equal(t, "PROMETHEUS", UpperValue("prometheus"))
	assert.Equal(t, "prometheus", LowerValue("PROMETHEUS"))
	assert.Empty(t, TrimValue(nil))
}

func TestWithPrefixValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "#alerts", WithPrefixValue("#", "alerts"))
	assert.Equal(t, "#alerts", WithPrefixValue("#", "#alerts"))
	assert.Equal(t, "#alerts", WithPrefixValue(" # ", " alerts "))
	assert.Empty(t, WithPrefixValue("#", nil))
	assert.Equal(t, "alerts", WithPrefixValue("", "alerts"))
}

func TestWithSuffixValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "path/", WithSuffixValue("/", "path"))
	assert.Equal(t, "path/", WithSuffixValue("/", "path/"))
	assert.Equal(t, "path/", WithSuffixValue(" / ", " path "))
	assert.Empty(t, WithSuffixValue("/", nil))
	assert.Equal(t, "path", WithSuffixValue("", "path"))
}

func TestDurationValue(t *testing.T) {
	t.Parallel()

	t.Run("formats duration value", func(t *testing.T) {
		t.Parallel()

		got, err := DurationValue(90 * time.Second)

		require.NoError(t, err)
		assert.Equal(t, "1m30s", got)
	})

	t.Run("formats duration pointer", func(t *testing.T) {
		t.Parallel()

		value := 90 * time.Second

		got, err := DurationValue(&value)

		require.NoError(t, err)
		assert.Equal(t, "1m30s", got)
	})

	t.Run("formats duration string", func(t *testing.T) {
		t.Parallel()

		got, err := DurationValue("90s")

		require.NoError(t, err)
		assert.Equal(t, "1m30s", got)
	})

	t.Run("rejects empty duration string", func(t *testing.T) {
		t.Parallel()

		_, err := DurationValue("")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "duration value must not be empty")
	})

	t.Run("rejects invalid duration", func(t *testing.T) {
		t.Parallel()

		_, err := DurationValue("bad")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse duration")
	})

	t.Run("rejects nil duration pointer", func(t *testing.T) {
		t.Parallel()

		var value *time.Duration

		_, err := DurationValue(value)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "duration value must not be nil")
	})

	t.Run("rejects unsupported value", func(t *testing.T) {
		t.Parallel()

		_, err := DurationValue(42)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "got int")
	})
}

func TestIsTruthyValue(t *testing.T) {
	t.Parallel()

	t.Run("detects falsey values", func(t *testing.T) {
		t.Parallel()

		var ptr *string
		empty := ""

		assert.False(t, isTruthyValue(nil))
		assert.False(t, isTruthyValue(ptr))
		assert.False(t, isTruthyValue(""))
		assert.False(t, isTruthyValue(&empty))
		assert.False(t, isTruthyValue(0))
		assert.False(t, isTruthyValue(false))
		assert.False(t, isTruthyValue([]string(nil)))
		assert.False(t, isTruthyValue([]string{}))
		assert.False(t, isTruthyValue(map[string]string{}))
	})

	t.Run("detects truthy values", func(t *testing.T) {
		t.Parallel()

		text := "value"

		assert.True(t, isTruthyValue(&text))
		assert.True(t, isTruthyValue("value"))
		assert.True(t, isTruthyValue(1))
		assert.True(t, isTruthyValue(true))
		assert.True(t, isTruthyValue([]string{"value"}))
		assert.True(t, isTruthyValue(map[string]string{"key": "value"}))
		assert.True(t, isTruthyValue(time.Time{}))
	})
}

func TestIsEmptyValue(t *testing.T) {
	t.Parallel()

	t.Run("detects empty values", func(t *testing.T) {
		t.Parallel()

		var ptr *string

		assert.True(t, isEmptyValue(nil))
		assert.True(t, isEmptyValue(ptr))
		assert.True(t, isEmptyValue(""))
		assert.True(t, isEmptyValue(0))
		assert.True(t, isEmptyValue(false))
		assert.True(t, isEmptyValue(time.Time{}))
		assert.True(t, isEmptyValue([]string(nil)))
		assert.True(t, isEmptyValue([]string{}))
		assert.True(t, isEmptyValue(map[string]string{}))
	})

	t.Run("detects non-empty values", func(t *testing.T) {
		t.Parallel()

		empty := ""
		text := "value"

		assert.False(t, isEmptyValue(&empty))
		assert.False(t, isEmptyValue(&text))
		assert.False(t, isEmptyValue("value"))
		assert.False(t, isEmptyValue(1))
		assert.False(t, isEmptyValue(true))
		assert.False(t, isEmptyValue(time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)))
		assert.False(t, isEmptyValue([]string{"value"}))
		assert.False(t, isEmptyValue(map[string]string{"key": "value"}))
	})
}

func TestStringValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "42", stringValue(42))
	assert.Empty(t, stringValue(nil))
}

func TestTimeValue(t *testing.T) {
	t.Parallel()

	t.Run("converts time value", func(t *testing.T) {
		t.Parallel()

		value := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

		got, err := timeValue(value)

		require.NoError(t, err)
		assert.True(t, got.Equal(value))
	})

	t.Run("converts time pointer", func(t *testing.T) {
		t.Parallel()

		value := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

		got, err := timeValue(&value)

		require.NoError(t, err)
		assert.True(t, got.Equal(value))
	})

	t.Run("converts rfc3339 string", func(t *testing.T) {
		t.Parallel()

		value := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

		got, err := timeValue(value.Format(time.RFC3339Nano))

		require.NoError(t, err)
		assert.True(t, got.Equal(value))
	})
}

func TestDurationValueInternal(t *testing.T) {
	t.Parallel()

	t.Run("converts duration value", func(t *testing.T) {
		t.Parallel()

		value := 90 * time.Second

		got, err := durationValue(value)

		require.NoError(t, err)
		assert.Equal(t, value, got)
	})

	t.Run("converts duration pointer", func(t *testing.T) {
		t.Parallel()

		value := 90 * time.Second

		got, err := durationValue(&value)

		require.NoError(t, err)
		assert.Equal(t, value, got)
	})

	t.Run("converts duration string", func(t *testing.T) {
		t.Parallel()

		got, err := durationValue("90s")

		require.NoError(t, err)
		assert.Equal(t, 90*time.Second, got)
	})

	t.Run("rejects unsupported value", func(t *testing.T) {
		t.Parallel()

		_, err := durationValue(errors.New("unsupported"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "got *errors.errorString")
	})
}
