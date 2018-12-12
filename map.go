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
	"errors"
	"fmt"
	"github.com/ansel1/merry"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Keys returns a slice of the keys in the map
func Keys(m map[string]interface{}) (keys []string) {
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// Merge returns a new map, which is the deep merge of the
// normalized values of v1 and v2.
//
// Values in v2 override values in v1.
//
// Slices are merged simply by adding any v2 values which aren't
// already in v1's slice.  This won't do anything fancy with
// slices that have duplicate values.  Order is ignored.  E.g.:
//
//    [5, 6, 7] + [5, 5, 5, 4] = [5, 6, 7, 4]
//
// The return value is a copy.  v1 and v2 are not modified.
func Merge(v1, v2 interface{}) interface{} {
	v1, _ = normalize(v1, true, true, true)
	v2, _ = normalize(v2, true, true, true)
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
// Values are normalized before being passed to the transformer function.
// Any maps and slices are passed to the transform function as the whole value
// first, then each child value of the map/slice is passed to the transform
// function.
//
// The value returned by the transformer will replace the original value.
//
// If the transform function returns a non-primitive value, it will recurse into the new value.
//
// If the transformer function returns the error ErrStop, the process will abort with no error.
func Transform(v interface{}, transformer func(in interface{}) (interface{}, error)) (interface{}, error) {
	v, err := transform(v, transformer)
	if err == ErrStop {
		return v, nil
	}
	return v, err
}

// ErrStop can be returned by transform functions to end recursion early.  The Transform function will
// not return an error.
var ErrStop = errors.New("stop")

func transform(v interface{}, transformer func(in interface{}) (interface{}, error)) (interface{}, error) {
	v, _ = normalize(v, false, true, false)
	var err error
	v, err = transformer(v)
	if err != nil {
		return v, err
	}
	// normalize again, in case the transformer function altered v
	v, _ = normalize(v, false, true, false)
	switch t := v.(type) {
	case map[string]interface{}:
		for key, value := range t {
			t[key], err = transform(value, transformer)
			if err != nil {
				break
			}
		}
	case []interface{}:
		for i, value := range t {
			t[i], err = transform(value, transformer)
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
	trace               *string
	parseDates          bool
	roundDates          time.Duration
	truncateDates       time.Duration
	stripTimeZone       bool
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

// ParseDates enables special processing for date values.  Contains typically marshals time.Time values
// to a string before comparison.  This means comparisons of time.Time values will not account for
// zero values, ignoring time zones, rounding times to a nearest precision, etc.
//
// When ParseDates is specified, after the values are normalized to strings, the code will attempt
// to parse any string values back into time.Time values.  This allows correct processing of
// the time.Time zero values.
//
// If rounding is > 0, times will be rounded.  If ignoreTimeZone is true, the location data will
// be stripped off the time values before comparison.
func ParseDates(rounding time.Duration, ignoreTimeZone bool) ContainsOption {
	return func(o *containsOptions) {
		o.parseDates = true
		o.roundDates = rounding
		o.stripTimeZone = ignoreTimeZone
	}
}

// TruncateDates is like ParseDates, but truncates time values rather than rounding them.
// If both options are specified, truncation will be applied first.
func TruncateDates(truncation time.Duration, ignoreTimeZone bool) ContainsOption {
	return func(o *containsOptions) {
		o.parseDates = true
		o.truncateDates = truncation
		o.stripTimeZone = ignoreTimeZone
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

// Trace sets `s` to a string describing the path to the values where containment was false.  Helps
// debugging why one value doesn't contain another.  Sample output:
//
//     -> v1: map[time:2017-03-03T14:08:30.097698864-05:00]
//     -> v2: map[time:0001-01-01T00:00:00Z]
//     -> "time"
//     --> v1: 2017-03-03T14:08:30.097698864-05:00
//     --> v2: 0001-01-01T00:00:00Z
//
// If `s` is nil, it does nothing.
func Trace(s *string) ContainsOption {
	return func(o *containsOptions) {
		o.trace = s
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
	ctx := containsCtx{}
	for _, o := range options {
		o(&ctx.containsOptions)
	}

	v1, _ = normalize(v1, false, true, true)
	v2, _ = normalize(v2, false, true, true)

	// when this option is set, we essentially undo the JSON-ification of date values, parsing
	// them back into time.Time.  We then do rounding, and can do omitempty style processing
	// on them.
	if ctx.parseDates {
		walkFunc := func(in interface{}) (i interface{}, e error) {
			if s, ok := in.(string); ok {
				t, err := time.Parse(time.RFC3339, s)
				if err == nil {
					if t.IsZero() && ctx.matchEmptyMapValues {
						return nil, nil
					}
					if ctx.truncateDates > 0 {
						t = t.Truncate(ctx.truncateDates)
					}
					if ctx.roundDates > 0 {
						t = t.Round(ctx.roundDates)
					}
					if ctx.stripTimeZone {
						t = t.UTC()
					}
					return t, nil
				}
			}
			return in, nil
		}
		var err error
		v1, err = Transform(v1, walkFunc)
		if err != nil {
			panic(err)
		}
		v2, err = Transform(v2, walkFunc)
		if err != nil {
			panic(err)
		}
	}

	b := contains(v1, v2, &ctx)
	if !b && ctx.trace != nil {
		ctx.prependPathComponent("v1")
		path := strings.Join(ctx.path, ".")
		ctx.path[0] = "v2"
		path2 := strings.Join(ctx.path, ".")
		if ctx.traceMsg == "" {
			ctx.traceMsg = "v1 does not equal v2"
		}
		s := fmt.Sprintf("%s\n%s -> %+v\n%s -> %+v", ctx.traceMsg, path, ctx.v1, path2, ctx.v2)
		*ctx.trace = s
	}
	return b
}

type containsCtx struct {
	path     []string
	v1, v2   interface{}
	traceMsg string
	containsOptions
}

func (c *containsCtx) prependPathComponent(s string) {
	c.path = append(c.path, "")
	copy(c.path[1:], c.path)
	c.path[0] = s
}

func contains(v1, v2 interface{}, opt *containsCtx) (b bool) {
	opt.v1 = v1
	opt.v2 = v2

	switch t1 := v1.(type) {
	case string:
		switch t2 := v2.(type) {
		case string:
			if opt.stringContains {
				return strings.Contains(t1, t2)
			}
			return v1 == v2
		default:
			opt.traceMsg = fmt.Sprintf(`v1 type %T does not match v1 type %T`, v1, v2)
			return false
		}
	case bool, nil, float64:
		return v1 == v2
	case time.Time:
		switch t2 := v2.(type) {
		case time.Time:
			return t1.Equal(t2) && t1.Location() == t2.Location()
		default:
			opt.traceMsg = fmt.Sprintf(`v1 type %T does not match v1 type %T`, v1, v2)
			return false
		}
	case map[string]interface{}:
		t2, isMap := v2.(map[string]interface{})
		if !isMap {
			// v1 is a map, but v2 isn't; v1 can't contain v2
			opt.traceMsg = fmt.Sprintf(`v1 type %T does not match v1 type %T`, v1, v2)
			return false
		}
		if len(t2) > len(t1) {
			// if t2 is bigger than t1, then t1 can't contain t2
			opt.traceMsg = `v2 has more keys than v1`
			return false
		}
	nextkey:
		for key, val2 := range t2 {
			// reset v1 and v2, which were changed by the calls to contains()
			opt.v1 = v1
			opt.v2 = v2

			val1, present := t1[key]
			if !present {
				opt.traceMsg = fmt.Sprintf(`key "%s" in v2 is not present in v1`, key)
				return false
			}

			if opt.matchEmptyMapValues {
				if val2 == nil {
					continue nextkey
				}

				type1 := reflect.TypeOf(val1)
				if type1 != nil && reflect.DeepEqual(reflect.Zero(type1).Interface(), val2) {
					continue nextkey
				}
			}

			if !contains(val1, val2, opt) {
				// tracks where we are in the structure, used for tracing
				opt.prependPathComponent(key)
				return false
			}
		}
		return true
	case []interface{}:
		switch t2 := v2.(type) {
		case bool, nil, string, float64:
			for _, el1 := range t1 {
				if el1 == v2 {
					return true
				}
			}
			opt.traceMsg = fmt.Sprintf(`v1 does not contain "%+v"`, v2)
			return false
		case []interface{}:
		Search:
			for _, val2 := range t2 {
				for _, value := range t1 {
					if contains(value, val2, opt) {
						continue Search
					}
				}
				// one of the values in v2 was not found in v1
				// reset v1 and v2, which were changed by the calls to contains()
				opt.v1 = v1
				opt.v2 = v2
				opt.traceMsg = fmt.Sprintf(`v1 does not contain "%+v"`, val2)
				return false
			}
			return true
		default:
			opt.traceMsg = fmt.Sprintf(`v1 type %T does not match v1 type %T`, v1, v2)
			return false
		}
	default:
		// since we deeply normalized both values, we should not hit this.
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
// Returns PathNotFoundError if the next key in the path is not found.
//
// Returns PathNotMapError if evaluating a key against a value which is not
// a map (e.g. a slice or a primitive value, against
// which we can't evaluate a key name).
//
// Returns IndexOutOfBoundsError if evaluating a slice index against a
// slice value, and the index is out of bounds.
//
// Returns PathNotSliceError if evaluating a slice index against a value which
// isn't a slice.
func Get(v interface{}, path string) (interface{}, error) {
	parsedPath, err := ParsePath(path)
	if err != nil {
		return nil, merry.Prepend(err, "Couldn't parse the path")
	}
	out := v
	for i, part := range parsedPath {
		switch t := part.(type) {
		case string:
			out, err = normalize(out, false, true, false)
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
			out, err = normalize(out, false, true, false)
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

// Empty returns true if v is nil, empty, or a zero value.
//
// If v is a pointer, it is empty if the pointer is nil or invalid, but not
// empty if it points to a value, even if that value is zero.  For example:
//
//     Empty(0)  // true
//     i := 0
//     Empty(&i) // false
//     Empty(Widget{}) // true, zero value
//     Empty(&Widget{}) // false, non-nil pointer
//
// Maps, slices, arrays, and channels are considered empty if their
// length is zero.
//
// Strings are empty if they contain nothing but whitespace.
func Empty(v interface{}) bool {
	switch t := v.(type) {
	case nil:
		return true
	case bool:
		return !t // false is empty
	case int:
		return t == 0
	case int8:
		return t == 0
	case int16:
		return t == 0
	case int32:
		return t == 0
	case int64:
		return t == 0
	case float32:
		return t == 0
	case float64:
		return t == 0
	case uint:
		return t == 0
	case uint8:
		return t == 0
	case uint16:
		return t == 0
	case uint32:
		return t == 0
	case uint64:
		return t == 0
	case complex64:
		return t == 0
	case complex128:
		return t == 0
	case uintptr:
		return t == 0
	case string:
		return len(strings.TrimSpace(t)) == 0
	case map[string]interface{}:
		return len(t) == 0
	case []interface{}:
		return len(t) == 0
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Invalid:
			return true
		case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
			return rv.Len() == 0
		case reflect.Func:
			return false
		case reflect.Struct:
			return reflect.DeepEqual(rv.Interface(), reflect.Zero(rv.Type()).Interface())
		case reflect.UnsafePointer:
			return false
		case reflect.Ptr:
			return !rv.IsValid() || rv.IsNil()
		default:
			panic(fmt.Sprintf("kind %v should have been handled before this", rv.Kind().String()))
		}
	}
}
