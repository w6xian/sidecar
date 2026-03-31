package checker

import (
	"fmt"
	"testing"
)

func TestCheckMap(t *testing.T) {

	checks := New(
		Int("int"),
		Int32("int32"),
		Int64("int64"),
		Uint("uint"),
		Uint32("uint32"),
		Uint64("uint64"),
		Float32("float32"),
		Float64("float64"),
		String("string"),
		String("def", DefaultValue("default")),
		Others("other", func(v any) (any, error) {
			fmt.Printf("other: %v\n", v)
			value, ok := v.([]string)
			if !ok {
				return nil, fmt.Errorf("checker %s type error, expect []string, got %T", "other", value)
			}
			return value, nil
		}),
	)
	// 9223372036854775807 = 1^63 - 1
	kv := map[string]any{
		"int":     0,
		"int32":   int32(32),
		"int64":   int64(64),
		"uint":    uint(0),
		"uint32":  uint32(32),
		"uint64":  uint64(64),
		"float32": float32(1.32),
		"float64": float64(0.64),
		"string":  "abcede",
		"def":     "a",
		"other":   []string{"a", "b", "c"},
	}

	kv, err := checks.CheckMap(kv)
	if err != nil {
		t.Errorf("CheckMap %v failed. err: %v", kv, err)
	}
	for k, v := range kv {
		fmt.Printf("%s: %v\n", k, v)
	}
}
