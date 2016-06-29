// A set of utility functions for working with maps.
// Generally, maps and slices of any kind will work, but performance
// is optimized for maps returned by json.Unmarshal(b, &interface{}).  If
// all the maps are map[string]interface{}, and all the slices are
// []interface{}, and all the rest of the values are primitives, then
// reflection is avoided.
package maps

import (
	"encoding/json"
	"github.com/ansel1/merry"
	"github.com/elgs/gosplitargs"
	"reflect"
	"strconv"
	"strings"
)

// return a slice of the keys in the map
func Keys(m map[string]interface{}) (keys []string) {
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// return a new map, which is the deep merge of m1 and m2.
// values in m2 override values in m1.
// This recurses into nested slices and maps
// Slices are merged simply by adding any m2 values which aren't
// already in m1's slice.  This won't do anything fancy with
// slices that have duplicate values.  Order is ignored.  E.g.:
//
//    [5, 6, 7] + [5, 5, 5, 4] = [5, 6, 7, 4]
//
// All values in the result will be normalized according to the following
// rules:
//
// 1. All maps with string keys will be converted into map[string]interface{}
// 2. All slices will be converted to []interface{}
// 3. All primitive numeric types will be converted into float64
// 4. All other types will not be converted, and will not be merged.  v2's value will just overwrite v1's
//
// New copies of all maps and slices are made, so v1 and v2 will not be modified.
func Merge(v1, v2 interface{}) interface{} {
	v1, _ = normalize(v1, true, false, true)
	v2, _ = normalize(v2, true, false, true)
	switch t1 := v1.(type) {
	case map[string]interface{}:
		if t2, isMap := v2.(map[string]interface{}); isMap {
			for key, value := range t2 {
				t1[key] = Merge(t1[key], value)
			}
			return t1
		}
	case []interface{}:
		if t2, isSlice := v2.([]interface{}); isSlice {
			orig := t1[:]
			for _, value := range t2 {
				if !sliceContains(orig, value) {
					t1 = append(t1, value)
				}
			}
			return t1
		}
	}
	return v2
}

func sliceContains(s []interface{}, v interface{}) bool {
	switch v.(type) {
	case string, float64, bool, nil:
		for _, value := range s {
			if value == v {
				return true
			}
		}
		return false
	}
	for _, value := range s {
		if reflect.DeepEqual(v, value) {
			return true
		}
	}
	return false
}

// Returns true if m1 contains all the key paths as m2, and
// the values at those paths are the equal.  I.E. returns
// true if m2 is a subset of m1.
// This will recurse into nested maps.
// When comparing to slice values, it will return true if
// slice 1 has at least one value which contains each of the
// values in slice 2.  It's kind of dumb though.  If slice 1
// contains a single value, say a big map, which contains *all*
// the values in slice 2, then this will return true.  In other words.
// when a match in slice 1 is found, that item is *not* removed from
// the search when matching the next value in slice 2.
// Examples:
//
//     {"color":"red"} contains {}
//     {"color":"red"} contains {"color":"red"}
//     {"color":"red","flavor":"beef"} contains {"color":"red"}
//     {"labels":{"color":"red","flavor":"beef"}} contains {"labels":{"flavor":"beef"}}
//     {"tags":["red","green","blue"]} contains {"tags":["red","green"]}

// This is what I mean about slice containment being a little simplistic:
//
//     {"resources":[{"type":"car","color":"red","wheels":4}]} contains {"resources":[{"type":"car"},{"color","red"},{"wheels":4}]}
//
// That will return true, despite there being 3 items in contained slice and only one item in the containing slice.  The
// one item in the containing slice matches each of the items in the contained slice.
func Contains(v1, v2 interface{}) bool {
	v1, _ = normalize(v1, false, false, false)
	v2, _ = normalize(v2, false, false, false)

	switch t1 := v1.(type) {
	case bool, nil, string, float64:
		return v1 == v2
	case map[string]interface{}:
		t2, isMap := v2.(map[string]interface{})
		if !isMap {
			// v1 is a map, but v2 isn't; v1 can't contain v2
			return false
		}
		if len(t2) > len(t1) {
			// if t2 is bigger than t1, then t1 can't contain t2
			return false
		}
		for key, val2 := range t2 {
			val1, present := t1[key]
			if !present || !Contains(val1, val2) {
				return false
			}
		}
		return true
	case []interface{}:
		t2, isSlice := v2.([]interface{})
		if !isSlice {
			// v1 is a slice, but v2 isn't; v1 can't contain v2
			return false
		}
		// first, normalize the values in v1, so we
		// don't re-normalize them in each loop
		t1copy := make([]interface{}, len(t1))
		for i, value := range t1 {
			t1copy[i], _ = normalize(value, false, false, false)
		}
		for _, val2 := range t2 {
			found := false
		Search:
			for _, value := range t1copy {
				if Contains(value, val2) {
					found = true
					break Search
				}
			}
			if !found {
				// one of the values in v2 was not found in v1
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(v1, v2)
	}
}

// returns true if trees share common key paths, but the values
// at those paths are not equal.
// i.e. if the two maps were merged, no values would be overwritten
// conflicts == !contains(v1, v2) && !excludes(v1, v2)
// conflicts == !contains(merge(v1, v2), v1)
func Conflicts(m1, m2 map[string]interface{}) bool {
	return !Contains(Merge(m1, m2), m1)
}

func normalize(v interface{}, makeCopies, doMarshaling, recurse bool) (v2 interface{}, err error) {
	v2 = v
	copied := false
	switch t := v.(type) {
	case bool, string, nil, float64:
		return
	case int:
		return float64(t), nil
	case int8:
		return float64(t), nil
	case int16:
		return float64(t), nil
	case int32:
		return float64(t), nil
	case int64:
		return float64(t), nil
	case float32:
		return float64(t), nil
	case uint:
		return float64(t), nil
	case uint8:
		return float64(t), nil
	case uint16:
		return float64(t), nil
	case uint32:
		return float64(t), nil
	case uint64:
		return float64(t), nil
	case map[string]interface{}, []interface{}:
		if !makeCopies && !recurse {
			return
		}
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
			copied = true
			m := make(map[string]interface{}, rv.Len())
			for _, v := range rv.MapKeys() {
				m[v.String()] = rv.MapIndex(v).Interface()
			}
			v2 = m
		} else if rv.Kind() == reflect.Slice {
			copied = true
			l := rv.Len()
			s := make([]interface{}, l)
			for i := 0; i < l; i++ {
				s[i] = rv.Index(i).Interface()
			}
			v2 = s
		} else if doMarshaling {
			// marshal/unmarshal
			var b []byte
			b, err = json.Marshal(v)
			if err != nil {
				return
			}
			v2 = nil
			err = json.Unmarshal(b, &v2)
			return
		}
	}
	if recurse || (makeCopies && !copied) {
		switch t := v2.(type) {
		case map[string]interface{}:
			var m map[string]interface{}
			if makeCopies && !copied {
				m = make(map[string]interface{}, len(t))
			} else {
				// modify in place
				m = t
			}
			v2 = m
			for key, value := range t {
				if recurse {
					if value, err = normalize(value, makeCopies, doMarshaling, recurse); err != nil {
						return
					}
				}
				m[key] = value
			}
		case []interface{}:
			var s []interface{}
			if makeCopies && !copied {
				s = make([]interface{}, len(t))
			} else {
				// modify in place
				s = t
			}
			v2 = s
			for i := 0; i < len(t); i++ {
				if recurse {
					if s[i], err = normalize(t[i], makeCopies, doMarshaling, recurse); err != nil {
						return
					}
				} else {
					s[i] = t[i]
				}
			}
		default:
			panic("Should be either a map or slice by now")
		}
	}

	return
}

// Recursively converts v1 into a tree of maps, slices, and primitives.
// The types in the result will be the types the json package uses for unmarshalling
// into interface{}.  The rules are:
//
// 1. All maps with string keys will be converted into map[string]interface{}
// 2. All slices will be converted to []interface{}
// 3. All primitive numeric types will be converted into float64
// 4. string, bool, and nil are unmodified
// 5. All other values will be converted into the above types by doing a json.Marshal and Unmarshal
//
// Values in v1 will be modified in place if possible
func Normalize(v1 interface{}) (interface{}, error) {
	return normalize(v1, false, true, true)
}

var PathNotFoundError = merry.New("Path not found")
var PathNotMapError = merry.New("Path not map")
var PathNotSliceError = merry.New("Path not slice")
var IndexOutOfBoundsError = merry.New("Index out of bounds")

// Extracts the value at path from v.
// Path is in the form:
//
//     response.things[2].color.red
//
// You can use `merry` to test the types of return errors:
//
//     _, err := maps.Get("","")
//     if merry.Is(err, maps.PathNotFoundError) {
//       ...
//
// `v` can be any primitive, map (must be keyed by string, but any value type), or slice, nested arbitrarily deep
func Get(v interface{}, path string) (interface{}, error) {
	parts, err := gosplitargs.SplitArgs(path, "\\.", false)
	if err != nil {
		return nil, merry.Prepend(err, "Couldn't parse the path")
	}
	out := v
	for i := 0; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if len(part) == 0 {
			continue
		}
		sliceIdx := -1
		// first check of the path part ends in an array index, like
		//
		//     tags[2]
		//
		// Extract the "2", and truncate the part to "tags"
		if bracketIdx := strings.Index(part, "["); bracketIdx > -1 && strings.HasSuffix(part, "]") {
			if idx, err := strconv.Atoi(part[bracketIdx+1 : len(part)-1]); err == nil {
				sliceIdx = idx
				part = part[0:bracketIdx]
			}
		}

		if part = strings.TrimSpace(part); len(part) > 0 {
			// map key
			out, _ = normalize(out, false, false, false)
			if m, ok := out.(map[string]interface{}); ok {
				var present bool
				if out, present = m[part]; !present {
					return nil, PathNotFoundError.WithMessagef("%s not found", strings.Join(parts[0:i+1], "."))
				}
			} else {
				errPath := strings.Join(parts[0:i], ".")
				if len(errPath) == 0 {
					errPath = "v"
				}
				return nil, PathNotMapError.WithMessagef("%s is not a map", errPath)
			}
		}
		if sliceIdx > -1 {
			// slice index
			out, _ = normalize(out, false, false, false)
			if s, ok := out.([]interface{}); ok {
				if l := len(s); l <= sliceIdx {
					return nil, IndexOutOfBoundsError.WithMessagef("Index out of bounds at %s (len = %v)", strings.Join(parts[0:i+1], "."), l)
				} else {
					out = s[sliceIdx]
				}
			} else {
				errPath := strings.Join(append(parts[0:i], part), ".")
				if len(errPath) == 0 {
					errPath = "v"
				}
				return nil, PathNotSliceError.WithMessagef("%s is not a slice", errPath)
			}
		}
	}
	return out, nil
}

// returns true if v is:
//
// 1. nil
// 2. an empty string
// 3. an empty slice
// 4. an empty map
// 5. an empty array
// 6. an empty channel
//
// returns false otherwise
func Empty(v interface{}) bool {
	// no op.  just means the value wasn't a type that supports Len()
	defer func() { recover() }()
	switch t := v.(type) {
	case bool, int, int8, int16, int32, int64, float32, float64, uint, uint8, uint16, uint32, uint64:
		return false
	case nil:
		return true
	case string:
		return len(strings.TrimSpace(t)) == 0
	case map[string]interface{}:
		return len(t) == 0
	case []interface{}:
		return len(t) == 0
	default:
		rv := reflect.ValueOf(v)
		if rv.IsNil() {
			// handle case of (*Widget)(nil)
			return true
		}
		if rv.Kind() == reflect.Ptr {
			return Empty(rv.Elem())
		}
		return reflect.ValueOf(v).Len() == 0
	}
}
