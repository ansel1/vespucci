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
}

// If v is a string key'd map of any kind, return a Map.
// If v is a slice of any kind, return a Slice.
// Otherwise, return v.
func Adapter(v interface{}) interface{} {
	switch t := v.(type) {
	case float64, bool, string, int, int8, int16, int32, float32, nil:
		return v
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
