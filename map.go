// A set of utility functions for working with maps.
// Generally, maps and slices of any kind will work, but performance
// is optimized for maps returned by json.Unmarshal(b, &interface{}).  If
// all the maps are map[string]interface{}, and all the slices are
// []interface{}, and all the rest of the values are primitives, then
// reflection is avoided.
package maps

import (
	"errors"
	"reflect"
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
	case float64, bool, string, int, int8, int16, int32, float32, nil:
		return v1 == v2
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
