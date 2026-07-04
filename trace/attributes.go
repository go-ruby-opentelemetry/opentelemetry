package trace

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
)

// attribute builds an OpenTelemetry attribute.KeyValue from a Ruby-style value.
// It mirrors the coercions opentelemetry-ruby performs when a Ruby object is
// used as an attribute value: the primitive scalar types and their
// homogeneous-array forms are represented natively, and anything else is
// stringified (as MRI does via #to_s).
func attr(key string, value any) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case bool:
		return attribute.Bool(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case []string:
		return attribute.StringSlice(key, v)
	case []bool:
		return attribute.BoolSlice(key, v)
	case []int64:
		return attribute.Int64Slice(key, v)
	case []float64:
		return attribute.Float64Slice(key, v)
	default:
		return attribute.String(key, fmt.Sprint(v))
	}
}

// attrs converts a Ruby-style attribute hash into a stable slice of KeyValues.
func attrs(m map[string]any) []attribute.KeyValue {
	kvs := make([]attribute.KeyValue, 0, len(m))
	for k, v := range m {
		kvs = append(kvs, attr(k, v))
	}
	return kvs
}

// AttributesToMap converts recorded OpenTelemetry attributes back into a
// Ruby-style hash of native Go values, mirroring how opentelemetry-ruby exposes
// a span's attributes.
func AttributesToMap(kvs []attribute.KeyValue) map[string]any {
	m := make(map[string]any, len(kvs))
	for _, kv := range kvs {
		m[string(kv.Key)] = kv.Value.AsInterface()
	}
	return m
}
