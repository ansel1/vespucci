// Package maps is a set of utility functions for working with maps.
// Generally, maps and slices of any kind will work, but performance
// is optimized for maps returned by json.Unmarshal(b, &interface{}).  If
// all the maps are map[string]interface{}, and all the slices are
// []interface{}, and all the rest of the values are primitives, then
// reflection is avoided.
package maps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ansel1/merry"
	"reflect"
	"strconv"
	"strings"
)

// Keys returns a slice of the keys in the map
func Keys(m map[string]interface{}) (keys []string) {
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// Merge returns a new map, which is the deep merge of m1 and m2.
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

// Transform applies a transformation function to each value in tree.
// Values are normalized before being passed to the transformer function, the
// equivalent of calling Normalize(Copies:false,Deep:false,Marshal:false).
// Any maps and slices are passed to the transform function as the whole value
// first, then each child value of the map/slice is passed to the transform
// function.
// The value returned by the transformer will replace the original value.
// If the transform function returns a map[string]interface{} or []interface{}, Transform()
// will recurse into them.
func Transform(v interface{}, transformer func(in interface{}) (interface{}, error)) (interface{}, error) {
	v, _ = normalize(v, false, false, false)
	var err error
	v, err = transformer(v)
	if err != nil {
		return v, err
	}
	switch t := v.(type) {
	case map[string]interface{}:
		for key, value := range t {
			t[key], err = Transform(value, transformer)
			if err != nil {
				break
			}
		}
	case []interface{}:
		for i, value := range t {
			t[i], err = Transform(value, transformer)
			if err != nil {
				break
			}
		}
	}
	return v, err
}

type containsOptions struct {
	stringContains      bool
	stringMatches       bool
	matchEmptyMapValues bool
}

// ContainsOption is an option which modifies the behavior of the Contains() function
type ContainsOption func(*containsOptions)

// EmptyMapValuesMatchAny is a ContainsOption which allows looser matching of map values.
// If set, when matching map entries, an entry in v2's map will match an entry in v1's map if:
//
// - the key matches AND
// - the value in v1 contains the value in v2
//   OR the value in v2 is nil
//   OR the value in v2 is the zero value of the type of v1's value
//
// This is convenient when testing whether a struct contains another struct.  Structs are normalized
// by marshalling them to JSON.  Fields which don't have the `omitempty` option will appear in the
// normalized v2 value as map keys with zero values.  Using this option will allow that to match.
//
// This option can also be used to test for the presence of keys in v1 without needing to test the value:
//
//     v1 := map[string]interface{}{"color":"blue"}
//     v2 := map[string]interface{}{"color":nil}
//     Contains(v1, v2)  // false
//     Contains(v1, v2, EmptyMapValuesMatchAny()) // true
//     v1 := map[string]interface{}{}
//     Contains(v1, v2, EmptyMapValuesMatchAny()) // false, because v1 doesn't have "color" key
//
// Another use is testing the general type of the value:
//
//     v1 := map[string]interface{}{"size":5}
//     v2 := map[string]interface{}{"size":0}
//     Contains(v1, v2)  // false
//     Contains(v1, v2, EmptyMapValuesMatchAny()) // true
//     v2 := map[string]interface{}{"size":""}
//     Contains(v1, v2, EmptyMapValuesMatchAny()) // false, because type of value doesn't match (v1: number, v2: string)
//
func EmptyMapValuesMatchAny() ContainsOption {
	return func(o *containsOptions) {
		o.matchEmptyMapValues = true
	}
}

// StringContains is a ContainsOption which uses strings.Contains(v1, v2) to test
// for string containment.
//
// Without this option, strings (like other primitive values) must match exactly.
//
//     Contains("brown fox", "fox") // false
//     Contains("brown fox", "fox", StringContains()) // true
func StringContains() ContainsOption {
	return func(o *containsOptions) {
		o.stringContains = true
	}
}

// Contains tests whether v1 "contains" v2.  The notion of containment
// is based on postgres' JSONB containment operators.
//
// A map v1 "contains" another map v2 if v1 has contains all the keys in v2, and
// if the values in v2 are contained by the corresponding values in v1.
//
//     {"color":"red"} contains {}
//     {"color":"red"} contains {"color":"red"}
//     {"color":"red","flavor":"beef"} contains {"color":"red"}
//     {"labels":{"color":"red","flavor":"beef"}} contains {"labels":{"flavor":"beef"}}
//     {"tags":["red","green","blue"]} contains {"tags":["red","green"]}
//
// A scalar value v1 contains value v2 if they are equal.
//
//     5 contains 5
//     "red" contains "red"
//
// A slice v1 contains a slice v2 if all the values in v2 are contained by at
// least one value in v1:
//
//     ["red","green"] contains ["red"]
//     ["red"] contains ["red","red","red"]
//     // In this case, the single value in v1 contains each of the values
//     // in v2, so v1 contains v2
//     [{"type":"car","color":"red","wheels":4}] contains [{"type":"car"},{"color","red"},{"wheels":4}]
//
// A slice v1 also can contain a *scalar* value v2:
//
//     ["red"] contains "red"
//
// A struct v1 contains a struct v2 if they are deeply equal (using reflect.DeepEquals)
func Contains(v1, v2 interface{}, options ...ContainsOption) bool {
	opt := containsOptions{}
	for _, o := range options {
		o(&opt)
	}
	return contains(v1, v2, opt)
}

func contains(v1, v2 interface{}, opt containsOptions) bool {
	v1, _ = normalize(v1, false, false, false)
	v2, _ = normalize(v2, false, false, false)

	switch t1 := v1.(type) {
	case string:
		switch t2 := v2.(type) {
		case string:
			if opt.stringContains {
				return strings.Contains(t1, t2)
			}
			return v1 == v2
		default:
			return v1 == v2
		}
	case bool, nil, float64:
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
	nextkey:
		for key, val2 := range t2 {
			val1, present := t1[key]
			if !present {
				return false
			}

			if opt.matchEmptyMapValues {
				if val2 == nil {
					break nextkey
				}
				type1 := reflect.TypeOf(val1)
				if type1 != nil && reflect.DeepEqual(reflect.Zero(type1).Interface(), val2) {
					break nextkey
				}
			}
			if !contains(val1, val2, opt) {
				return false
			}
		}
		return true
	case []interface{}:
		switch t2 := v2.(type) {
		case bool, nil, string, float64:
			for _, el1 := range t1 {
				el1, _ = normalize(el1, false, false, false)
				if el1 == v2 {
					return true
				}
			}
			return false
		case []interface{}:
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
					if contains(value, val2, opt) {
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
			return false
		}
	default:
		return reflect.DeepEqual(v1, v2)
	}
}

// Conflicts returns true if trees share common key paths, but the values
// at those paths are not equal.
// i.e. if the two maps were merged, no values would be overwritten
// conflicts == !contains(v1, v2) && !excludes(v1, v2)
// conflicts == !contains(merge(v1, v2), v1)
func Conflicts(m1, m2 map[string]interface{}) bool {
	return !Contains(Merge(m1, m2), m1)
}

// NormalizeOptions are options for the Normalize function.
type NormalizeOptions struct {
	// Make copies of all maps and slices.  The result will not share
	// any maps or slices with input value.
	Copy bool

	// if values are encountered which are not primitives, maps, or slices, attempt to
	// turn them into primitives, maps, and slices by running through json.Marshal and json.Unmarshal
	Marshal bool

	// Perform the operation recursively.  If false, only v is normalized, but nested values are not
	Deep bool
}

// NormalizeWithOptions does the same as Normalize, but with options.
func NormalizeWithOptions(v interface{}, opt NormalizeOptions) (interface{}, error) {
	return normalize(v, opt.Copy, opt.Marshal, opt.Deep)
}

func normalize(v interface{}, copies, marshal, deep bool) (v2 interface{}, err error) {
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
		if !copies && !deep {
			return
		}
	default:
		// if v explicitly supports json marshalling, just skip to that.
		if marshal {
			switch m := v.(type) {
			case json.Marshaler:
				return slowNormalize(m)
			case json.RawMessage:
				// This handles a special case for golang < 1.8
				// Below 1.8, *json.RawMessage implemented json.Marshaler, but
				// json.Marshaler did not (weird, since it's based on a slice type, so
				// it can already be nil)
				// This was fixed in 1.8, so as of 1.8, we'll never hit this case (the
				// first case will be hit)
				return slowNormalize(&m)
			}
		}
		rv := reflect.ValueOf(v)
		switch {
		case rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String:
			copied = true
			m := make(map[string]interface{}, rv.Len())
			for _, v := range rv.MapKeys() {
				m[v.String()] = rv.MapIndex(v).Interface()
			}
			v2 = m
		case rv.Kind() == reflect.Slice:
			copied = true
			l := rv.Len()
			s := make([]interface{}, l)
			for i := 0; i < l; i++ {
				s[i] = rv.Index(i).Interface()
			}
			v2 = s
		case marshal:
			// marshal/unmarshal
			return slowNormalize(v)
		default:
			// return value unchanged
			return
		}
	}
	if deep || (copies && !copied) {
		switch t := v2.(type) {
		case map[string]interface{}:
			var m map[string]interface{}
			if copies && !copied {
				m = make(map[string]interface{}, len(t))
			} else {
				// modify in place
				m = t
			}
			v2 = m
			for key, value := range t {
				if deep {
					if value, err = normalize(value, copies, marshal, deep); err != nil {
						return
					}
				}
				m[key] = value
			}
		case []interface{}:
			var s []interface{}
			if copies && !copied {
				s = make([]interface{}, len(t))
			} else {
				// modify in place
				s = t
			}
			v2 = s
			for i := 0; i < len(t); i++ {
				if deep {
					if s[i], err = normalize(t[i], copies, marshal, deep); err != nil {
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

func slowNormalize(v interface{}) (interface{}, error) {
	var b []byte
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var v2 interface{}
	err = json.Unmarshal(b, &v2)
	return v2, err
}

// Normalize recursively converts v1 into a tree of maps, slices, and primitives.
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
	return normalize(v1, true, true, true)
}

// PathNotFoundError indicates the requested path was not present in the value.
var PathNotFoundError = merry.New("Path not found")

// PathNotMapError indicates the value at the path is not a map.
var PathNotMapError = merry.New("Path not map")

// PathNotSliceError indicates the value at the path is not a slice.
var PathNotSliceError = merry.New("Path not slice")

// IndexOutOfBoundsError indicates the index doesn't exist in the slice.
var IndexOutOfBoundsError = merry.New("Index out of bounds")

// Path is a slice of either strings or slice indexes (ints).
type Path []interface{}

// ParsePath parses a string path into a Path slice.  String paths look
// like:
//
//     user.name.first
//     user.addresses[3].street
//
func ParsePath(path string) (Path, error) {
	var parsedPath Path
	parts := strings.Split(path, ".")
	for i := 0; i < len(parts); i++ {
		part := parts[i]

		arrayIdx := -1
		// first check of the path part ends in an array index, like
		//
		//     tags[2]
		//
		// Extract the "2", and truncate the part to "tags"
		if bracketIdx := strings.Index(part, "["); bracketIdx > -1 && strings.HasSuffix(part, "]") {
			if idx, err := strconv.Atoi(part[bracketIdx+1 : len(part)-1]); err == nil {
				arrayIdx = idx
				part = part[0:bracketIdx]
			}
		}

		part = strings.TrimSpace(part)
		if len(part) > 0 {
			parsedPath = append(parsedPath, part)
		}
		if arrayIdx > -1 {
			parsedPath = append(parsedPath, arrayIdx)
		}
	}
	return parsedPath, nil
}

// String implements the Stringer interface.  It returns the string
// representation of a Path.  Path.String() and ParsePath() are inversions
// of each other.
func (p Path) String() string {
	buf := bytes.NewBuffer(nil)

	for _, elem := range p {
		switch t := elem.(type) {
		case string:
			if buf.Len() > 0 {
				buf.WriteString(".")
			}
			buf.WriteString(t)
		case int:
			if strings.HasSuffix(buf.String(), "]") {
				buf.WriteString(".")
			}
			fmt.Fprintf(buf, "[%d]", t)
		default:
			panic(merry.Errorf("Path element was not a string or int! elem: %#v", elem))
		}
	}
	return buf.String()
}

// GetOptions are options to the Get operation.
// Currently an alias for NormalizeOptions (the value is normalized before
// the path is evaluated against it), but don't count on this alias.
type GetOptions NormalizeOptions

// Get extracts the value at path from v.
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
	return GetWithOpts(v, path, GetOptions{})
}

// GetWithOpts is like Get, but with options.
func GetWithOpts(v interface{}, path string, opts GetOptions) (interface{}, error) {
	parsedPath, err := ParsePath(path)
	if err != nil {
		return nil, merry.Prepend(err, "Couldn't parse the path")
	}
	out := v
	for i, part := range parsedPath {
		switch t := part.(type) {
		case string:
			out, err = NormalizeWithOptions(out, NormalizeOptions(opts))
			if err != nil {
				return nil, err
			}
			if m, ok := out.(map[string]interface{}); ok {
				var present bool
				if out, present = m[t]; !present {
					return nil, PathNotFoundError.Here().WithMessagef("%v not found", parsedPath[0:i+1])
				}
			} else {
				if i > 0 {
					return nil, PathNotMapError.Here().WithMessagef("%v is not a map", parsedPath[0:i])
				}
				return nil, PathNotMapError.Here().WithMessage("v is not a map")
			}
		case int:
			// slice index
			out, err = NormalizeWithOptions(out, NormalizeOptions(opts))
			if err != nil {
				return nil, err
			}
			if s, ok := out.([]interface{}); ok {
				if l := len(s); l <= t {
					return nil, IndexOutOfBoundsError.Here().WithMessagef("Index out of bounds at %v (len = %v)", parsedPath[0:i+1], l)
				}
				out = s[t]
			} else {
				if i > 0 {
					return nil, PathNotSliceError.Here().WithMessagef("%v is not a slice", parsedPath[0:i])
				}
				return nil, PathNotSliceError.Here().WithMessage("v is not a slice")
			}
		default:
			panic(merry.Errorf("Unexpected type for parsed path element: %#v", part))
		}
	}
	return out, nil
}

// Empty returns true if v is:
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
	defer func() {
		recover()
	}()
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
