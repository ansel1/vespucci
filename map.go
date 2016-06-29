// A set of utility functions for working with maps.
// Generally, maps and slices of any kind will work, but performance
// is optimized for maps returned by json.Unmarshal(b, &interface{}).  If
// all the maps are map[string]interface{}, and all the slices are
// []interface{}, and all the rest of the values are primitives, then
// reflection is avoided.
package maps

import (
	"encoding/json"
	"errors"
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
// The type of all maps in the returned result will be
// map[string]interface{}, and the type of all slices in the result
// will be []interface{} (regardless of the types of the maps and slices
// in the arguments.
func Merge(m1, m2 map[string]interface{}) map[string]interface{} {
	if m1 == nil && m2 == nil {
		return nil
	}
	r := map[string]interface{}{}
	for key, value := range m1 {
		r[key] = value
	}
	for key, value := range m2 {
		r[key] = mergeMapValues(r[key], value)
	}
	return r
}

var stop = errors.New("")

func mergeMapValues(v1, v2 interface{}) interface{} {
	switch t1 := Adapter(v1).(type) {
	case Map:
		if t2, ok := Adapter(v2).(Map); ok {
			r := map[string]interface{}{}
			t1.Visit(func(k string, v interface{}) error {
				r[k] = v
				return nil
			})
			t2.Visit(func(k string, v interface{}) error {
				r[k] = mergeMapValues(r[k], v)
				return nil
			})
			return r
		}
	case Slice:
		if t2, ok := Adapter(v2).(Slice); ok {
			var comb []interface{}
			t1.Visit(func(_ int, v interface{}) error {
				comb = append(comb, v)
				return nil
			})
			t2.Visit(func(_ int, v interface{}) error {
				comb = mergeValueIntoSlice(v, comb)
				return nil
			})
			return comb
		}
	}
	return v2
}

func mergeValueIntoSlice(v interface{}, dst []interface{}) []interface{} {
	switch v.(type) {
	case string, int, int8, int16, int32, float64, float32, bool, nil:
		for _, v2 := range dst {
			if v == v2 {
				return dst
			}
		}
	default:
		for _, rv := range dst {
			if reflect.DeepEqual(rv, v) {
				// value is already in the dst slice.  Skip it.
				// todo: not clear what the best behavior is here.  slices are not sets, so uniqueness is not implied
				// but generally, this is probably what developers intend when they "merge" two slices.
				// also, if both slices contain other slices or maps, they won't be merged.  Again, not sure
				// how you pair up value to merge anyway, but hopefully this simple implementation is sufficient
				// for now.
				return dst
			}
		}
	}
	return append(dst, v)
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
	switch t1 := Adapter(v1).(type) {
	case bool, nil, string:
		return v1 == v2
	case float64:
		if t2, ok := Adapter(v2).(float64); ok {
			return t1 == t2
		}
	case Map:
		if t2, ok := Adapter(v2).(Map); ok {
			if t2.Len() > t1.Len() {
				// if t2 is bigger than t1, then t1 can't contain t2
				return false
			}

			e := t2.Visit(func(k string, vv2 interface{}) error {
				vv1, present := t1.Get(k)
				if !present || !Contains(vv1, vv2) {
					return stop
				}
				return nil
			})
			return e == nil
		}
	case Slice:
		if t2, ok := Adapter(v2).(Slice); ok {
			e := t2.Visit(func(i int, v2 interface{}) error {
				e := t1.Visit(func(_ int, v1 interface{}) error {
					if Contains(v1, v2) {
						return stop
					}
					return nil
				})
				if e == nil {
					// one t2's values was not found in t1.  t1 doesn't contain t2
					return stop
				}
				return nil
			})
			return e != stop
		}
	}
	return reflect.DeepEqual(v1, v2)
}

// returns true if trees share common key paths, but the values
// at those paths are not equal.
// i.e. if the two maps were merged, no values would be overwritten
// conflicts == !contains(v1, v2) && !excludes(v1, v2)
// conflicts == !contains(merge(v1, v2), v1)
func Conflicts(m1, m2 map[string]interface{}) bool {
	return !Contains(Merge(m1, m2), m1)
}

func normalize(v1 interface{}) (v2 interface{}, converted bool, err error) {
	// handle all the basic types we don't need to convert
	v2 = v1
	switch t := v1.(type) {
	case bool, nil, string, float64, int, int8, int16, int32, int64, float32:
		return
	case []bool, []string, []float32, []float64, []int, []int8, []int16, []int32, []int64:
		return
	case map[string]string, map[string]bool, map[string]float32, map[string]float64, map[string]int, map[string]int8, map[string]int16, map[string]int32, map[string]int64:
		return
	case map[string]interface{}:
		// recurse
		for key, value := range t {
			var vv interface{}
			var conv bool
			vv, conv, err = normalize(value)
			if err != nil {
				return
			}
			if conv {
				t[key] = vv
				converted = true
			}
		}
	case []interface{}:
		// recurse
		for i := 0; i < len(t); i++ {
			var vv interface{}
			var conv bool
			vv, conv, err = normalize(t[i])
			if err != nil {
				return
			}
			if conv {
				t[i] = vv
				converted = true
			}
		}
	default:
		// marshal/unmarshal
		converted = true
		var b []byte
		b, err = json.Marshal(v1)
		if err != nil {
			return
		}
		v2 = nil
		err = json.Unmarshal(b, &v2)
	}
	return
}

// Converts any value into a structure of nested maps, slices, and primitives.
// Any values which aren't one of those are converted by marshaling and unmarshaling
// the value through the json package.
// Effectively the same as using the json package to marshal and then unmarshal
// the value, but can be faster, as it will traverse the object and avoid the
// marshalling dance if the value is already a map, slice, or primitive.
func Normalize(v1 interface{}) (v2 interface{}, err error) {
	v2, _, err = normalize(v1)
	return
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
			if m, ok := Adapter(out).(Map); ok {
				var present bool
				if out, present = m.Get(part); !present {
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
			if s, ok := Adapter(out).(Slice); ok {
				if l := s.Len(); l <= sliceIdx {
					return nil, IndexOutOfBoundsError.WithMessagef("Index out of bounds at %s (len = %v)", strings.Join(parts[0:i+1], "."), l)
				} else {
					out = s.Get(sliceIdx)
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
