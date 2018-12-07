package maps

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ansel1/merry"
	"github.com/k0kubun/pp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sort"
	"testing"
	"time"
)

func TestMerge(t *testing.T) {
	jsonTests := []struct {
		src, dst, expected string
	}{
		{`{"hot":"weather"}`, `{"hard":"lemonade"}`, `{"hot":"weather","hard":"lemonade"}`},
		{`{"hot":"weather"}`, `{"hot":"roof"}`, `{"hot":"weather"}`},
		{`{"colors":{"color":"orange"}}`, `{"hot":"roof"}`, `{"hot":"roof","colors":{"color":"orange"}}`},
		{`{"colors":{"warm":"orange"}, "numbers":{"odd":3, "even":4}}`, `{"hot":"roof", "colors":{"warm":"red","cool":"blue"}, "flavors":{"sweet":"chocolate"}}`, `{"hot":"roof", "colors":{"warm":"orange","cool":"blue"}, "flavors":{"sweet":"chocolate"}, "numbers":{"odd":3, "even":4}}`},
		{`{"tags":["east","blue"]}`, `{"tags":["east","loud","big"]}`, `{"tags":["east", "loud", "big", "blue"]}`},
	}
	for _, test := range jsonTests {
		dst := toMap(test.dst)
		r := Merge(dst, toMap(test.src))
		assert.Equal(t, toMap(test.expected), r)
	}

	literalMapsTests := []struct {
		m1, m2, m3 map[string]interface{}
	}{
		{
			map[string]interface{}{"tags": []string{"red", "green"}},
			map[string]interface{}{"tags": []string{"green", "blue"}},
			map[string]interface{}{"tags": []interface{}{"red", "green", "blue"}},
		},
		{
			map[string]interface{}{"color": "red"},
			map[string]interface{}{"color": "blue"},
			map[string]interface{}{"color": "blue"},
		},
	}
	for _, test := range literalMapsTests {
		r := Merge(test.m1, test.m2)
		assert.Equal(t, test.m3, r)
	}

	// make sure v1 is not modified
	m1 := map[string]interface{}{"color": "blue"}
	m2 := map[string]interface{}{"color": "red"}
	m3 := Merge(m1, m2)
	assert.Equal(t, map[string]interface{}{"color": "red"}, m3)
	assert.Equal(t, map[string]interface{}{"color": "blue"}, m1)
}

func TestKeys(t *testing.T) {
	tests := []struct {
		m map[string]interface{}
		k []string
	}{
		{map[string]interface{}{"color": "blue", "price": "high", "weight": 2}, []string{"color", "price", "weight"}},
	}
	for _, test := range tests {
		sort.Strings(test.k)
		out := Keys(test.m)
		sort.Strings(out)
		assert.Equal(t, test.k, out)
	}
}

func bigNestedMaps(prefix string, nesting int) map[string]interface{} {
	r := map[string]interface{}{}
	for i := 0; i < 2; i++ {
		r[fmt.Sprintf("%vtop%v", prefix, i)] = fmt.Sprintf("topval%v", i)
		r[fmt.Sprintf("%vtopint%v", prefix, i)] = i
		r[fmt.Sprintf("%vtopbool%v", prefix, i)] = true
		if nesting > 0 {
			r[fmt.Sprintf("%vnested%v", prefix, i)] = bigNestedMaps(prefix, nesting-1)
		}
	}
	return r
}

func TestContains(t *testing.T) {
	t1 := time.Now()

	tests := []struct {
		name     string
		v1, v2   interface{}
		expected bool
		options  []ContainsOption
	}{
		{v1: "red", v2: "red", expected: true},
		{v1: "red", v2: "green"},
		{
			v1: []string{"big", "loud"},
			v2: []string{"smart", "loud"},
		},
		{
			v1:       []string{"big", "loud"},
			v2:       []string{"big"},
			expected: true,
		},
		{
			v1:       []string{"big", "loud", "high"},
			v2:       []string{"big", "loud"},
			expected: true,
		},
		{
			v1: []string{"big", "loud", "high"},
			v2: []string{"big", "rough"},
		},
		{
			v1:       []string{"red", "green"},
			v2:       "red",
			expected: true,
		},
		{
			v1: []string{"red", "green"},
			v2: "blue",
		},
		{
			v1: []string{"red", "green"},
			v2: map[string]interface{}{"red": "green"},
		},
		{
			v1: map[string]interface{}{"resource": map[string]interface{}{"id": 1, "color": "red", "tags": []string{"big", "loud"}}, "environment": map[string]interface{}{"time": "night", "source": "east"}},
			v2: map[string]interface{}{"resource": map[string]interface{}{"tags": []string{"smart", "loud"}}},
		},
		{
			v1:       map[string]interface{}{"color": "red"},
			v2:       map[string]interface{}{"color": "red"},
			expected: true,
		},
		{
			v1: map[string]interface{}{"color": "green"},
			v2: map[string]interface{}{"color": "red"},
		},
		{
			v1: map[string]interface{}{"color": "green"},
			v2: "color",
		},
		{
			expected: true,
		},
		{
			v2:       "red",
			expected: false,
		},
		{
			v1:       "red",
			v2:       "red",
			expected: true,
		},
		{
			v1: "red",
			v2: "green",
		},
		{
			v1:       true,
			v2:       true,
			expected: true,
		},
		{
			v1: true,
			v2: false,
		},
		{
			v1:       5,
			v2:       float64(5),
			expected: true,
		},
		{
			v1:       map[string]interface{}{"color": "green", "flavor": "beef"},
			v2:       map[string]interface{}{"color": "green"},
			expected: true,
		},
		{
			v1:       map[string]interface{}{"color": "green", "tags": []string{"beef", "hot"}},
			v2:       map[string]interface{}{"color": "green", "tags": []string{"hot"}},
			expected: true,
		},
		{
			v1: map[string]interface{}{"color": "green", "tags": []string{"beef", "hot"}},
			v2: map[string]interface{}{"color": "green", "tags": []string{"cool"}},
		},
		{
			v1: map[string]interface{}{
				"resource": map[string]interface{}{
					"id":    1,
					"color": "red",
					"size":  6,
					"labels": map[string]interface{}{
						"region": "east",
						"level":  "high",
					},
					"tags": []string{"trouble", "up", "down"},
				},
				"principal": map[string]interface{}{
					"name":   "bob",
					"role":   "admin",
					"groups": []string{"officers", "gentlemen"},
				},
			},
			v2: map[string]interface{}{
				"resource": map[string]interface{}{
					"color": "red",
					"size":  6,
					"labels": map[string]interface{}{
						"region": "east",
					},
					"tags": []interface{}{"up"},
				},
				"principal": map[string]interface{}{
					"role":   "admin",
					"groups": []interface{}{"officers"},
				},
			},
			expected: true,
		},
		{
			v1: map[string]interface{}{
				"resource": map[string]interface{}{
					"id":    1,
					"color": "red",
					"size":  6,
					"labels": map[string]interface{}{
						"region": "east",
						"level":  "high",
					},
					"tags": []string{"trouble", "up", "down"},
				},
				"principal": map[string]interface{}{
					"name":   "bob",
					"role":   "admin",
					"groups": []string{"officers", "gentlemen"},
				},
			},
			v2: map[string]interface{}{
				"resource": map[string]interface{}{
					"size": 7,
				},
			},
		},
		{
			v1:       "The quick brown fox",
			v2:       "quick brown",
			expected: false,
		},
		{
			v1:       "The quick brown fox",
			v2:       "quick brown",
			options:  []ContainsOption{StringContains()},
			expected: true,
		},
		{
			v1:       map[string]interface{}{"story": "The quick brown fox"},
			v2:       map[string]interface{}{"story": "quick brown"},
			expected: false,
		},
		{
			v1:       map[string]interface{}{"story": "The quick brown fox"},
			v2:       map[string]interface{}{"story": "quick brown"},
			options:  []ContainsOption{StringContains()},
			expected: true,
		},
		{
			v1:       []string{"The quick brown fox"},
			v2:       []string{"quick brown"},
			expected: false,
		},
		{
			v1:       []string{"The quick brown fox"},
			v2:       []string{"quick brown"},
			options:  []ContainsOption{StringContains()},
			expected: true,
		},
		{
			v1:       map[string]interface{}{"color": "blue"},
			v2:       map[string]interface{}{"color": ""},
			expected: false,
		},
		{
			v1:       map[string]interface{}{"color": "blue"},
			v2:       map[string]interface{}{"color": nil},
			expected: false,
		},
		{
			v1:       map[string]interface{}{"color": "blue"},
			v2:       map[string]interface{}{"color": ""},
			options:  []ContainsOption{EmptyMapValuesMatchAny()},
			expected: true,
		},
		{
			v1:       map[string]interface{}{"color": "blue"},
			v2:       map[string]interface{}{"color": nil},
			options:  []ContainsOption{EmptyMapValuesMatchAny()},
			expected: true,
		},
		{
			name:     "emptymapvaluemustmatchtype",
			v1:       map[string]interface{}{"color": "blue"},
			v2:       map[string]interface{}{"color": 0},
			options:  []ContainsOption{EmptyMapValuesMatchAny()},
			expected: false,
		},
		{
			name:     "emptydate",
			v1:       map[string]interface{}{"t": time.Now()},
			v2:       map[string]interface{}{"t": time.Time{}},
			options:  []ContainsOption{EmptyMapValuesMatchAny()},
			expected: false,
		},
		{
			name:     "parseDate",
			v1:       map[string]interface{}{"t": time.Now()},
			v2:       map[string]interface{}{"t": time.Time{}},
			options:  []ContainsOption{EmptyMapValuesMatchAny(), ParseDates(0, true)},
			expected: true,
		},
		{
			name:     "equaldates",
			v1:       map[string]interface{}{"t": t1},
			v2:       map[string]interface{}{"t": t1},
			options:  []ContainsOption{EmptyMapValuesMatchAny(), ParseDates(0, true)},
			expected: true,
		},
		{
			name:     "unequaldates",
			v1:       map[string]interface{}{"t": t1},
			v2:       map[string]interface{}{"t": t1.Add(time.Nanosecond)},
			options:  []ContainsOption{EmptyMapValuesMatchAny(), ParseDates(0, true)},
			expected: false,
		},
		{
			name:     "rounddates",
			v1:       map[string]interface{}{"t": t1},
			v2:       map[string]interface{}{"t": t1.Add(time.Nanosecond)},
			options:  []ContainsOption{EmptyMapValuesMatchAny(), ParseDates(time.Microsecond, true)},
			expected: true,
		},
		{
			name:     "timezones",
			v1:       map[string]interface{}{"t": t1.In(time.FixedZone("test", -3))},
			v2:       map[string]interface{}{"t": t1.UTC()},
			options:  []ContainsOption{EmptyMapValuesMatchAny(), ParseDates(0, false)},
			expected: false,
		},
		{
			name:     "timezonesutc",
			v1:       map[string]interface{}{"t": t1},
			v2:       map[string]interface{}{"t": t1.UTC()},
			options:  []ContainsOption{EmptyMapValuesMatchAny(), ParseDates(0, true)},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, Contains(test.v1, test.v2, test.options...), pp.Sprintln("m1", test.v1, "m2", test.v2))
		})
	}

	t.Run("trace", func(t *testing.T) {
		v1 := map[string]interface{}{"color": "red"}
		v2 := map[string]interface{}{"color": "red"}
		var trace string
		Contains(v1, v2, Trace(&trace))
		assert.Empty(t, trace, "trace should be empty if contains returned true")
		v2["color"] = "blue"
		Contains(v1, v2, Trace(&trace))
		assert.NotEmpty(t, trace, "trace should not be empty if contains returned false")
		t.Log(trace)
		assert.NotPanics(t, func() {
			Contains(v1, v2, Trace(nil))
		})

		tests := []struct {
			v1, v2        interface{}
			expectedTrace string
		}{
			{v1: 1, v2: 2, expectedTrace: `
v1 does not equal v2
v1 -> 1
v2 -> 2`},
			{v1: map[string]string{"color": "red"}, v2: 1, expectedTrace: `
v1 type map[string]interface {} does not match v1 type float64
v1 -> map[color:red]
v2 -> 1`},
			{v1: map[string]string{"color": "red"}, v2: map[string]string{"color": "blue"}, expectedTrace: `
v1 does not equal v2
v1.color -> red
v2.color -> blue`},
			{v1: map[string]interface{}{"color": map[string]string{"height": "tall"}}, v2: map[string]interface{}{"color": map[string]string{"height": "short"}}, expectedTrace: `
v1 does not equal v2
v1.color.height -> tall
v2.color.height -> short`},
			{v1: map[string]interface{}{"color": "blue"}, v2: map[string]interface{}{"size": "big"}, expectedTrace: `
key "size" in v2 is not present in v1
v1 -> map[color:blue]
v2 -> map[size:big]`},
		}
		for _, test := range tests {
			t.Run("", func(t *testing.T) {
				var trace string
				Contains(test.v1, test.v2, Trace(&trace))
				t.Log(trace)
				// strip off the leading new line.  Only there to make the test more readable
				assert.Equal(t, test.expectedTrace[1:], trace)
			})
		}
	})

}

func TestConflicts(t *testing.T) {
	tests := []struct {
		m1, m2   map[string]interface{}
		expected bool
	}{
		{
			map[string]interface{}{"color": "red"},
			map[string]interface{}{"temp": "hot"},
			false,
		},
		{
			map[string]interface{}{"color": "red"},
			map[string]interface{}{"color": "blue"},
			true,
		},
		{
			map[string]interface{}{"color": "red"},
			map[string]interface{}{"temp": "hot", "color": "red"},
			false,
		},
		{
			map[string]interface{}{
				"labels": map[string]interface{}{"region": "west"},
			},
			map[string]interface{}{
				"temp":   "hot",
				"labels": map[string]interface{}{"region": "west"},
			},
			false,
		},
		{
			map[string]interface{}{
				"labels": map[string]interface{}{"region": "east"},
			},
			map[string]interface{}{
				"temp":   "hot",
				"labels": map[string]interface{}{"region": "west"},
			},
			true,
		},
		{
			map[string]interface{}{
				"tags": []string{"green", "red"},
			},
			map[string]interface{}{
				"tags": []string{"orange", "black"},
			},
			false,
		},
		{
			map[string]interface{}{
				"tags": []string{"green", "red"},
			},
			map[string]interface{}{
				"tags": []string{"orange", "black", "red"},
			},
			false,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, Conflicts(test.m1, test.m2), pp.Sprintln("m1", test.m1, "m2", test.m2))
		// should be reflexive
		assert.Equal(t, test.expected, Conflicts(test.m2, test.m1), pp.Sprintln("inverse m1", test.m1, "m2", test.m2))

	}
}

func BenchmarkBigMerge(b *testing.B) {
	m1 := map[string]interface{}{}
	m1["matches"] = bigNestedMaps("color", 5)
	m1["notmatches"] = bigNestedMaps("weather", 5)
	s1 := []interface{}{}
	for i := 0; i < 100; i++ {
		s1 = append(s1, bigNestedMaps(fmt.Sprintf("food%v", i), 3))
	}
	m1["slice"] = s1
	m2 := map[string]interface{}{}
	m2["matches"] = bigNestedMaps("color", 5)
	m2["notmatches"] = bigNestedMaps("cars", 5)
	s2 := []interface{}{}
	for i := 0; i < 100; i++ {
		s2 = append(s2, bigNestedMaps(fmt.Sprintf("water%v", i), 3))
	}
	//pp.Println("m1", m1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Merge(m1, m2)
	}
}

func BenchmarkMerge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Merge(map[string]interface{}{
			"colors": map[string]interface{}{
				"warm": "orange",
			},
			"numbers": map[string]interface{}{
				"odd":  3,
				"even": 4,
			},
		}, map[string]interface{}{
			"hot": "roof",
			"colors": map[string]interface{}{
				"warm": "red",
				"cool": "blue",
			},
			"flavors": map[string]interface{}{
				"sweet": "chocolate",
			},
		},
		)
	}
}

func toMap(s string) (out map[string]interface{}) {
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		panic(err)
	}
	return
}

type Widget struct {
	Size  int    `json:"size"`
	Color string `json:"color"`
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		in, out interface{}
		opts    *NormalizeOptions
	}{
		// basic no-op types of cases
		{in: 5, out: float64(5)},
		{in: "red", out: "red"},
		{},
		{in: float64(10), out: float64(10)},
		{in: float32(12), out: float64(12)},
		{in: true, out: true},
		{in: map[string]interface{}{"red": "green"}, out: map[string]interface{}{"red": "green"}},
		{in: []interface{}{"red", 4}, out: []interface{}{"red", float64(4)}},
		{in: []string{"red", "green"}, out: []interface{}{"red", "green"}},
		// hits the marshaling path
		{in: &Widget{5, "red"}, out: map[string]interface{}{"size": float64(5), "color": "red"}},
		// marshaling might occur deep
		{in: map[string]interface{}{"widget": &Widget{5, "red"}}, out: map[string]interface{}{"widget": map[string]interface{}{"size": float64(5), "color": "red"}}},
		{
			name: "marshaller",
			in:   json.RawMessage(`{"color":"blue"}`),
			out:  map[string]interface{}{"color": "blue"},
			opts: &NormalizeOptions{Marshal: true},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out interface{}
			var err error
			if test.opts != nil {
				out, err = NormalizeWithOptions(test.in, *test.opts)
			} else {
				out, err = Normalize(test.in)
			}

			require.NoError(t, err)
			require.Equal(t, test.out, out, "in: %v", pp.Sprint(test.in))
		})
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		v, out interface{}
		path   string
	}{
		{5, 5, ""},
		{[]string{"red"}, "red", "[0]"},
		{map[string]interface{}{"color": "red"}, "red", "color"},
		{map[string]interface{}{"tags": []string{"red", "green"}}, "green", "tags[1]"},
		{map[string]interface{}{"tags": []string{"red", "green"}}, "red", "tags[0]"},
		{map[string]interface{}{"resource": map[string]interface{}{"tags": []string{"red", "green"}}}, "red", "resource.tags[0]"},
		{map[string]interface{}{"resource": map[string]interface{}{"tags": []string{"red", "green"}}}, []string{"red", "green"}, "resource.tags"},
		{map[string]interface{}{"resource": map[string]interface{}{"tags": []string{"red", "green"}}}, map[string]interface{}{"tags": []string{"red", "green"}}, "resource"},
	}
	for _, test := range tests {
		result, err := Get(test.v, test.path)
		require.NoError(t, err, "v = %#v, path = %v", test.v, test.path)
		require.Equal(t, test.out, result, "v = %#v, path = %v", test.v, test.path)
	}

	// errors
	errorTests := []struct {
		v         interface{}
		path, msg string
		kind      error
	}{
		{map[string]interface{}{"tags": []string{"red", "green"}}, "tags[2]", "Index out of bounds at tags[2] (len = 2)", IndexOutOfBoundsError},
		{[]string{"red", "green"}, "[2]", "Index out of bounds at [2] (len = 2)", IndexOutOfBoundsError},
		{map[string]interface{}{"tags": "red"}, "tags[2]", "tags is not a slice", PathNotSliceError},
		{map[string]interface{}{"tags": "red"}, "[2]", "v is not a slice", PathNotSliceError},
		{[]string{"red", "green"}, "tags[2]", "v is not a map", PathNotMapError},
		{map[string]interface{}{"tags": "red"}, "color", "color not found", PathNotFoundError},
	}
	for _, test := range errorTests {
		_, err := Get(test.v, test.path)
		assert.EqualError(t, err, test.msg, "v = %#v, path = %v", test.v, test.path)
		assert.True(t, merry.Is(err, test.kind), "Wrong type of error.  Expected %v, was %v", test.kind, err)
	}
}

type holder struct {
	i interface{}
}

func TestEmpty(t *testing.T) {
	var ptr *Widget
	var nilPtr *Widget
	nilPtr = nil
	emptyTests := []interface{}{
		nil, "", "  ", map[string]interface{}{},
		[]interface{}{}, map[string]string{},
		[]string{},
		ptr,
		(*Widget)(nil),
		nilPtr,
		[0]string{},
		false,
		0,
		int8(0),
		int16(0),
		int32(0),
		int64(0),
		float32(0),
		float64(0),
		uint(0),
		uint8(0),
		uint16(0),
		uint32(0),
		uint64(0),
		uintptr(0),
		complex64(0),
		complex128(0),
		Widget{},

		make(chan string),
		holder{},
	}
	for _, v := range emptyTests {
		assert.True(t, Empty(v), "v = %#v", v)
	}
	notEmptyTests := []interface{}{
		5, float64(5), true,
		"asdf", " asdf ", map[string]interface{}{"color": "red"},
		map[string]string{"color": "red"}, []interface{}{"color", "red"},
		[]string{"green", "red"}, &Widget{Color: "blue"}, Widget{Color: "red"},
		func() {}, &Widget{},
		holder{i: Widget{}},
	}
	for _, v := range notEmptyTests {
		assert.False(t, Empty(v), "v = %#v", v)
	}
}

func BenchmarkEmpty(b *testing.B) {
	var w Widget
	for i := 0; i < b.N; i++ {
		Empty(w)
	}
}

func TestInternalNormalize(t *testing.T) {
	tm := time.Now()
	v, err := normalize(tm, false, false, true)
	assert.NoError(t, err)
	assert.Equal(t, v, tm)
}

func TestTransform(t *testing.T) {
	in := map[string]interface{}{
		"color": "red",
		"size":  5,
		"tags": []interface{}{
			"blue",
			false,
			nil,
		},
		"labels": map[string]interface{}{
			"region": "east",
		},
	}
	transformer := func(in interface{}) (interface{}, error) {
		if s, ok := in.(string); ok {
			return s + "s", nil
		}
		return in, nil
	}
	expected := map[string]interface{}{
		"color": "reds",
		"size":  float64(5),
		"tags": []interface{}{
			"blues",
			false,
			nil,
		},
		"labels": map[string]interface{}{
			"region": "easts",
		},
	}
	out, err := Transform(in, transformer)
	assert.NoError(t, err)
	assert.Equal(t, expected, out)

	// errors float out
	transformer = func(in interface{}) (interface{}, error) {
		if in == nil {
			return nil, errors.New("stop")
		}
		return in, nil
	}
	_, err = Transform(out, transformer)
	assert.EqualError(t, err, "stop")

	// I can transform the maps and slices at the top level too
	transformer = func(in interface{}) (interface{}, error) {
		switch t := in.(type) {
		case map[string]interface{}:
			delete(t, "labels")
		case []interface{}:
			in = append(t, "dogs")
		}
		return in, nil
	}

	expected = map[string]interface{}{
		"color": "reds",
		"size":  float64(5),
		"tags": []interface{}{
			"blues",
			false,
			nil,
			"dogs",
		},
	}
	out, err = Transform(out, transformer)
	assert.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		in           string
		out          Path
		checkReverse bool
	}{
		{"", nil, true},
		{"a", Path{"a"}, true},
		{"a.b", Path{"a", "b"}, true},
		{"a.b..c", Path{"a", "b", "c"}, false},
		{"[3]", Path{3}, true},
		{"a[3]", Path{"a", 3}, true},
		{"a.b[3]", Path{"a", "b", 3}, true},
		{"a[1].b[3]", Path{"a", 1, "b", 3}, true},
		{"[1].[3]", Path{1, 3}, true},
		{"a[b].c", Path{"a[b]", "c"}, true},
	}
	for _, test := range tests {
		out, err := ParsePath(test.in)
		assert.NoError(t, err)
		assert.Equal(t, test.out, out, "input: %v", test.in)
		if test.checkReverse {
			assert.Equal(t, test.in, out.String(), "testing conversion back to string")
		}
	}

	assert.Equal(t, "a.b[3]", Path{"a", "b", 3, "c", 4}[0:3].String())
}

func BenchmarkMarshal(b *testing.B) {
	s := struct {
		Map   map[string]string
		Slice []string
	}{
		Map: map[string]string{
			"color": "blue",
		},
		Slice: []string{"green", "yellow"},
	}
	for i := 0; i < b.N; i++ {
		Normalize(s)
	}
}

func BenchmarkAvoidMarshal(b *testing.B) {
	s := map[string]interface{}{
		"Map": map[string]string{
			"color": "blue",
		},
		"Slice": []string{"green", "yellow"},
	}
	for i := 0; i < b.N; i++ {
		Normalize(s)
	}
}
