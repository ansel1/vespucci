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
	"sort"
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
	return merge(v1, v2)
}

func merge(v1, v2 interface{}) interface{} {
	switch t1 := v1.(type) {
	case map[string]interface{}:
		if t2, isMap := v2.(map[string]interface{}); isMap {
			for key, value := range t2 {
				t1[key] = merge(t1[key], value)
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
	stringContains   bool
	matchEmptyValues bool
	trace            *string
	parseTimes       bool
	roundTimes       time.Duration
	truncateTimes    time.Duration
	timeDelta        time.Duration
	ignoreTimeZone   bool
	invertValues     bool
}

// ContainsOption is an option which modifies the behavior of the Contains() function
type ContainsOption func(*containsOptions)

// EmptyMapValuesMatchAny is an alias for EmptyValuesMatchAny.
var EmptyMapValuesMatchAny = EmptyValuesMatchAny

// EmptyValuesMatchAny is a ContainsOption which allows looser matching of empty values.
// If set, a value in v1 will match a value in v2 if:
//
// - v1 contains v2
// - OR v2 is nil
// - OR v2 is the zero value of the type of v1's value
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
func EmptyValuesMatchAny() ContainsOption {
	return func(o *containsOptions) {
		o.matchEmptyValues = true
	}
}

// ParseTimes enables special processing for date values.  Contains typically marshals time.Time values
// to a string before comparison.  This means the EmptyValuesMatchAny() option will not work
// as expected for time values.
//
// When ParseTimes is specified, after the values are normalized to strings, the code will attempt
// to parse any string values back into time.Time values.  This allows correct processing of
// the time.Time zero values.
func ParseTimes() ContainsOption {
	return func(o *containsOptions) {
		o.parseTimes = true
	}
}

// AllowTimeDelta configures the precision of time comparison.  Time values will be considered equal if the
// difference between the two values is less than d.
//
// Implies ParseTimes
func AllowTimeDelta(d time.Duration) ContainsOption {
	return func(o *containsOptions) {
		o.parseTimes = true
		o.timeDelta = d
	}
}

// TruncateTimes will truncate time values (see time.Time#Truncate)
//
// Implies ParseTimes
func TruncateTimes(d time.Duration) ContainsOption {
	return func(o *containsOptions) {
		o.parseTimes = true
		o.truncateTimes = d
	}
}

// RoundTimes will round time values (see time.Time#Round)
//
// Implies ParseTimes
func RoundTimes(d time.Duration) ContainsOption {
	return func(o *containsOptions) {
		o.parseTimes = true
		o.roundTimes = d
	}
}

// IgnoreTimeZones will ignore the time zones of time values (otherwise
// the time zones must match).
//
// Implies ParseTimes
func IgnoreTimeZones(b bool) ContainsOption {
	return func(o *containsOptions) {
		o.parseTimes = true
		o.ignoreTimeZone = true
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
	result, _, _ := ContainsEx(v1, v2, options...)
	return result
}

// ContainsEx is the same as Contains, but returns the normalized versions of v1 and v2 used
// in the comparison.
func ContainsEx(v1, v2 interface{}, options ...ContainsOption) (result bool, v1Norm, v2Norm interface{}) {
	ctx := containsCtx{}
	for _, o := range options {
		o(&ctx.containsOptions)
	}

	v1, _ = normalize(v1, false, true, true)
	v2, _ = normalize(v2, false, true, true)

	return contains(v1, v2, &ctx), v1, v2
}

// Equivalent checks if v1 and v2 are approximately deeply equal to each other.
// It takes the same comparison options as Contains.  It is equivalent to:
//
//     Equivalent(v1, v2) == Contains(v1, v2) && Contains(v2, v1)
//
// ContainsOptions which only work in one direction, like StringContains, will
// always treat v2 as a pattern or rule to match v1 against.  For example:
//
//     b := Equivalent("thefox", "fox", StringContains())
//
// b is true because "thefox" contains "fox", even though the inverse is not true
func Equivalent(v1, v2 interface{}, options ...ContainsOption) bool {
	result, _, _ := EquivalentEx(v1, v2, options...)
	return result
}

// EquivalentEx is the same as Equivalent, but returns the normalized versions of v1 and v2 used
// in the comparison.
func EquivalentEx(v1, v2 interface{}, options ...ContainsOption) (result bool, v1Norm, v2Norm interface{}) {
	ctx := containsCtx{}
	for _, o := range options {
		o(&ctx.containsOptions)
	}

	v1, _ = normalize(v1, false, true, true)
	v2, _ = normalize(v2, false, true, true)

	if !contains(v1, v2, &ctx) {
		return false, v1, v2
	}

	ctx.invertValues = true
	return contains(v2, v1, &ctx), v1, v2
}

type containsCtx struct {
	path []string
	containsOptions
}

func (c *containsCtx) traceMsg(msg string, v1, v2 interface{}) {
	if c.trace == nil {
		return
	}

	path1 := strings.Join(append([]string{"v1"}, c.path...), "")
	path2 := strings.Join(append([]string{"v2"}, c.path...), "")

	if c.invertValues {
		msg = strings.ReplaceAll(msg, "v1", path2)
		msg = strings.ReplaceAll(msg, "v2", path1)

		*c.trace = fmt.Sprintf("%s\n%s -> %+v\n%s -> %+v", msg, path1, v2, path2, v1)
	} else {
		msg = strings.ReplaceAll(msg, "v1", path1)
		msg = strings.ReplaceAll(msg, "v2", path2)

		*c.trace = fmt.Sprintf("%s\n%s -> %+v\n%s -> %+v", msg, path1, v1, path2, v2)
	}

}

func (c *containsCtx) traceNotEqual(v1, v2 interface{}) {
	c.traceMsg("values are not equal", v1, v2)
}

func contains(v1, v2 interface{}, opt *containsCtx) (b bool) {
	if opt.matchEmptyValues {
		match := v1
		against := v2
		if opt.invertValues {
			match = v2
			against = v1
		}
		if against == nil {
			return true
		}

		type1 := reflect.TypeOf(match)
		if type1 != nil && reflect.DeepEqual(reflect.Zero(type1).Interface(), against) {
			return true
		}
	}

	switch t1 := v1.(type) {
	case string:
		if v1 == v2 {
			return true
		}

		s2, ok := v2.(string)
		if !ok {
			opt.traceNotEqual(v1, v2)
			return false
		}

		if opt.parseTimes {
			tm1, err := time.Parse(time.RFC3339Nano, t1)
			if err == nil {
				tm2, err := time.Parse(time.RFC3339Nano, s2)
				if err == nil {
					if opt.matchEmptyValues {
						if opt.invertValues && tm1.IsZero() {
							return true
						} else if tm2.IsZero() {
							return true
						}
					}
					if opt.truncateTimes > 0 {
						tm1 = tm1.Truncate(opt.truncateTimes)
						tm2 = tm2.Truncate(opt.truncateTimes)
					}
					if opt.roundTimes > 0 {
						tm1 = tm1.Round(opt.roundTimes)
						tm2 = tm2.Round(opt.roundTimes)
					}
					delta := tm1.Sub(tm2)
					if delta < 0 {
						delta *= -1
					}
					if delta > opt.timeDelta {
						if opt.timeDelta > 0 {
							opt.traceMsg(fmt.Sprintf(`delta of %v exceeds %v`, delta, opt.timeDelta), v1, v2)
						} else {
							opt.traceNotEqual(v1, v2)
						}
						return false
					}
					if opt.ignoreTimeZone {
						return true
					}
					if tm1.Location() != tm2.Location() {
						opt.traceMsg(`time zone offsets don't match`, v1, v2)
					}
					return true
				}
			}
		}
		if opt.stringContains {
			match := t1
			against := s2
			if opt.invertValues {
				match = s2
				against = t1
			}
			if !strings.Contains(match, against) {
				opt.traceMsg(`v1 does not contain v2`, v1, v2)
			}
			return true
		}
		opt.traceNotEqual(v1, v2)
		return false
	case bool, nil, float64:
		if v1 != v2 {
			opt.traceNotEqual(v1, v2)
			return false
		}
		return true
	case map[string]interface{}:
		t2, ok := v2.(map[string]interface{})
		if !ok {
			// v1 is a map, but v2 isn't; v1 can't contain v2
			opt.traceNotEqual(v1, v2)
			return false
		}
		var extraKeys []string
		for key, val2 := range t2 {
			val1, present := t1[key]
			if !present {
				extraKeys = append(extraKeys, key)
			} else {
				opt.path = append(opt.path, "."+key)
				b2 := contains(val1, val2, opt)
				opt.path = opt.path[:len(opt.path)-1]
				if !b2 {
					return false
				}
			}
		}
		if len(extraKeys) > 0 {
			sort.Strings(extraKeys)
			opt.traceMsg(fmt.Sprintf(`v2 contains extra keys: %v`, extraKeys), v1, v2)
			return false
		}
		return true
	case []interface{}:
		switch t2 := v2.(type) {
		default:
			for _, el1 := range t1 {
				if contains(el1, v2, opt) {
					return true
				}
			}
			opt.traceMsg(fmt.Sprintf(`v1 does not contain v2`), v1, v2)
			return false
		case []interface{}:
		Search:
			for i, val2 := range t2 {
				for _, value := range t1 {
					if contains(value, val2, opt) {
						continue Search
					}
				}
				opt.traceMsg(fmt.Sprintf(`v1 does not contain v2[%v]: "%+v"`, i, val2), v1, v2)
				return false
			}
			return true
		}
	default:
		// since we deeply normalized both values, we should not hit this.
		equal := reflect.DeepEqual(v1, v2)
		if !equal {
			opt.traceMsg("values are not equal", v1, v2)
		}
		return equal
	}
}

// Conflicts returns true if trees share common key paths, but the values
// at those paths are not equal.
// i.e. if the two maps were merged, no values would be overwritten
// conflicts == !contains(v1, v2) && !excludes(v1, v2)
// conflicts == !contains(merge(v1, v2), v1)
func Conflicts(v1, v2 interface{}) bool {
	v1, _ = normalize(v1, true, true, true)
	v2, _ = normalize(v2, true, true, true)

	// merge will modify v1 in place.  make a pristine copy of v1 to compare against the merge
	v1orig, _ := normalize(v1, true, false, true)
	return !contains(merge(v1, v2), v1orig, &containsCtx{})
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
