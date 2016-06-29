package maps

import "reflect"

// A map with string keys.
type Map interface {
	// func is called for each key,value pair in the map
	Visit(func(key string, val interface{}) error) error
	Len() int
	// returns the value at the key, and whether the key is present in the map
	Get(key string) (interface{}, bool)
}

// A slice
type Slice interface {
	// func is called for each value in the slice
	Visit(func(i int, val interface{}) error) error
	Len() int
	Get(i int) interface{}
}

// If v is a string key'd map of any kind, return a Map.
// If v is a slice of any kind, return a Slice.
// If v is a number primitive, convert to float64
// Otherwise, return v.
func Adapter(v interface{}) interface{} {
	switch t := v.(type) {
	case bool, string, nil, float64:
		return v
	case int:
		return float64(t)
	case int8:
		return float64(t)
	case int16:
		return float64(t)
	case int32:
		return float64(t)
	case int64:
		return float64(t)
	case float32:
		return float64(t)
	case map[string]interface{}:
		return jsonObj(t)
	case []interface{}:
		return jsonArray(t)
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
			return reflectMap(rv)
		}
		if rv.Kind() == reflect.Slice {
			return reflectSlice(rv)
		}
	}
	return v
}
