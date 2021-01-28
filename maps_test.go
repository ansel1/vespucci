package maps

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ansel1/merry"
	"github.com/ansel1/vespucci/v4/proto"
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
		m1, m2, m3 dict
	}{
		{
			dict{"tags": []string{"red", "green"}},
			dict{"tags": []string{"green", "blue"}},
			dict{"tags": []interface{}{"red", "green", "blue"}},
		},
		{
			dict{"color": "red"},
			dict{"color": "blue"},
			dict{"color": "blue"},
		},
	}
	for _, test := range literalMapsTests {
		r := Merge(test.m1, test.m2)
		assert.Equal(t, test.m3, r)
	}

	// make sure v1 is not modified
	m1 := dict{"color": "blue"}
	m2 := dict{"color": "red"}
	m3 := Merge(m1, m2)
	assert.Equal(t, dict{"color": "red"}, m3)
	assert.Equal(t, dict{"color": "blue"}, m1)
}

func TestKeys(t *testing.T) {
	tests := []struct {
		m dict
		k []string
	}{
		{dict{"color": "blue", "price": "high", "weight": 2}, []string{"color", "price", "weight"}},
	}
	for _, test := range tests {
		sort.Strings(test.k)
		out := Keys(test.m)
		sort.Strings(out)
		assert.Equal(t, test.k, out)
	}
}

func bigNestedMaps(prefix string, nesting int) dict {
	r := dict{}
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
			v2: dict{"red": "green"},
		},
		{
			v1: dict{"resource": dict{"id": 1, "color": "red", "tags": []string{"big", "loud"}}, "environment": dict{"time": "night", "source": "east"}},
			v2: dict{"resource": dict{"tags": []string{"smart", "loud"}}},
		},
		{
			v1:       dict{"color": "red"},
			v2:       dict{"color": "red"},
			expected: true,
		},
		{
			v1: dict{"color": "green"},
			v2: dict{"color": "red"},
		},
		{
			v1: dict{"color": "green"},
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
			v1:       dict{"color": "green", "flavor": "beef"},
			v2:       dict{"color": "green"},
			expected: true,
		},
		{
			v1:       dict{"color": "green", "tags": []string{"beef", "hot"}},
			v2:       dict{"color": "green", "tags": []string{"hot"}},
			expected: true,
		},
		{
			v1: dict{"color": "green", "tags": []string{"beef", "hot"}},
			v2: dict{"color": "green", "tags": []string{"cool"}},
		},
		{
			v1: dict{
				"resource": dict{
					"id":    1,
					"color": "red",
					"size":  6,
					"labels": dict{
						"region": "east",
						"level":  "high",
					},
					"tags": []string{"trouble", "up", "down"},
				},
				"principal": dict{
					"name":   "bob",
					"role":   "admin",
					"groups": []string{"officers", "gentlemen"},
				},
			},
			v2: dict{
				"resource": dict{
					"color": "red",
					"size":  6,
					"labels": dict{
						"region": "east",
					},
					"tags": []interface{}{"up"},
				},
				"principal": dict{
					"role":   "admin",
					"groups": []interface{}{"officers"},
				},
			},
			expected: true,
		},
		{
			v1: dict{
				"resource": dict{
					"id":    1,
					"color": "red",
					"size":  6,
					"labels": dict{
						"region": "east",
						"level":  "high",
					},
					"tags": []string{"trouble", "up", "down"},
				},
				"principal": dict{
					"name":   "bob",
					"role":   "admin",
					"groups": []string{"officers", "gentlemen"},
				},
			},
			v2: dict{
				"resource": dict{
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
			v1:       dict{"story": "The quick brown fox"},
			v2:       dict{"story": "quick brown"},
			expected: false,
		},
		{
			v1:       dict{"story": "The quick brown fox"},
			v2:       dict{"story": "quick brown"},
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
			v1:       dict{"color": "blue"},
			v2:       dict{"color": ""},
			expected: false,
		},
		{
			v1:       dict{"color": "blue"},
			v2:       dict{"color": nil},
			expected: false,
		},
		{
			v1:       dict{"color": "blue"},
			v2:       dict{"color": ""},
			options:  []ContainsOption{EmptyMapValuesMatchAny()},
			expected: true,
		},
		{
			v1:       dict{"color": "blue"},
			v2:       dict{"color": nil},
			options:  []ContainsOption{EmptyMapValuesMatchAny()},
			expected: true,
		},
		{
			name:     "emptymapvaluemustmatchtype",
			v1:       dict{"color": "blue"},
			v2:       dict{"color": 0},
			options:  []ContainsOption{EmptyMapValuesMatchAny()},
			expected: false,
		},
		{
			name:     "emptyvaluematchnil",
			v1:       "blue",
			v2:       nil,
			options:  []ContainsOption{EmptyValuesMatchAny()},
			expected: true,
		},
		{
			name:     "emptyvaluematchzero",
			v1:       "blue",
			v2:       "",
			options:  []ContainsOption{EmptyValuesMatchAny()},
			expected: true,
		},
		{
			name:     "notemptyvalue",
			v1:       "blue",
			v2:       "red",
			options:  []ContainsOption{EmptyValuesMatchAny()},
			expected: false,
		},
		{
			name:     "emptydate",
			v1:       t1,
			v2:       time.Time{},
			options:  []ContainsOption{EmptyValuesMatchAny()},
			expected: false,
		},
		{
			name:     "parseTimes",
			v1:       t1,
			v2:       time.Time{},
			options:  []ContainsOption{EmptyValuesMatchAny(), ParseTimes()},
			expected: true,
		},
		{
			name:     "equaldates",
			v1:       t1,
			v2:       t1,
			options:  []ContainsOption{ParseTimes()},
			expected: true,
		},
		{
			name:     "unequaldates",
			v1:       t1,
			v2:       t1.Add(time.Nanosecond),
			options:  []ContainsOption{ParseTimes()},
			expected: false,
		},
		{
			name:     "truncatedates",
			v1:       t1.Truncate(time.Microsecond).Add(900 * time.Nanosecond),
			v2:       t1.Truncate(time.Microsecond).Add(100 * time.Nanosecond),
			options:  []ContainsOption{TruncateTimes(time.Microsecond)},
			expected: true,
		},
		{
			name:     "rounddates",
			v1:       t1.Truncate(time.Microsecond).Add(900 * time.Nanosecond),
			v2:       t1.Truncate(time.Microsecond).Add(100 * time.Nanosecond),
			options:  []ContainsOption{RoundTimes(time.Microsecond)},
			expected: false,
		},
		{
			name:     "rounddates",
			v1:       t1.Truncate(time.Microsecond).Add(900 * time.Nanosecond),
			v2:       t1.Truncate(time.Microsecond).Add(700 * time.Nanosecond),
			options:  []ContainsOption{RoundTimes(time.Microsecond)},
			expected: true,
		},
		{
			name:     "timezones",
			v1:       t1.In(time.FixedZone("test", -3)),
			v2:       t1.UTC(),
			options:  []ContainsOption{},
			expected: false,
		},
		{
			name:     "timezones",
			v1:       t1.In(time.FixedZone("test", -3)),
			v2:       t1.UTC(),
			options:  []ContainsOption{ParseTimes()},
			expected: false,
		},
		{
			name:     "timezones",
			v1:       t1.In(time.FixedZone("test", -3)),
			v2:       t1.UTC(),
			options:  []ContainsOption{IgnoreTimeZones(false)},
			expected: false,
		},
		{
			name:     "timezonesutc",
			v1:       t1.In(time.FixedZone("test", -3*60*60)),
			v2:       t1.UTC(),
			options:  []ContainsOption{IgnoreTimeZones(true)},
			expected: true,
		},
		{
			name:     "allowtimedelta",
			v1:       t1,
			v2:       t1.Add(time.Microsecond),
			options:  []ContainsOption{AllowTimeDelta(time.Microsecond)},
			expected: true,
		},
		{
			name:     "allowtimedelta",
			v1:       t1,
			v2:       t1.Add(time.Microsecond),
			options:  []ContainsOption{AllowTimeDelta(time.Microsecond / 2)},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, Contains(test.v1, test.v2, test.options...), pp.Sprintln("m1", test.v1, "m2", test.v2))
		})
	}

	t.Run("trace", func(t *testing.T) {
		v1 := dict{"color": "red"}
		v2 := dict{"color": "red"}
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

		now := time.Date(1987, 2, 10, 6, 30, 15, 0, time.FixedZone("EST", -5*60*60))
		nowCST := now.In(time.FixedZone("CST", -6*60*60))

		tests := []struct {
			v1, v2        interface{}
			expectedTrace string
			opts          []ContainsOption
		}{
			{v1: 1, v2: 2, expectedTrace: `
values are not equal
v1 -> 1
v2 -> 2`},
			{v1: "red", v2: "blue", expectedTrace: `
values are not equal
v1 -> red
v2 -> blue`},
			{v1: "red", v2: 1, expectedTrace: `
values are not equal
v1 -> red
v2 -> 1`},
			{v1: true, v2: false, expectedTrace: `
values are not equal
v1 -> true
v2 -> false`},
			{v1: true, v2: nil, expectedTrace: `
values are not equal
v1 -> true
v2 -> <nil>`},
			{v1: nil, v2: false, expectedTrace: `
values are not equal
v1 -> <nil>
v2 -> false`},
			{v1: float64(1), v2: false, expectedTrace: `
values are not equal
v1 -> 1
v2 -> false`},
			{v1: float64(1), v2: float64(2), expectedTrace: `
values are not equal
v1 -> 1
v2 -> 2`},
			{v1: "red", v2: "blue", opts: []ContainsOption{StringContains()}, expectedTrace: `
v1 does not contain v2
v1 -> red
v2 -> blue`},
			{v1: dict{"color": "red"}, v2: 1, expectedTrace: `
values are not equal
v1 -> map[color:red]
v2 -> 1`},
			{v1: dict{"color": "red"}, v2: dict{"color": "blue"}, expectedTrace: `
values are not equal
v1.color -> red
v2.color -> blue`},
			{v1: dict{"color": dict{"height": "tall"}}, v2: dict{"color": dict{"height": "short"}}, expectedTrace: `
values are not equal
v1.color.height -> tall
v2.color.height -> short`},
			{v1: dict{"color": "blue"}, v2: dict{"color": "blue", "size": "big", "flavor": "strawberry"}, expectedTrace: `
v2 contains extra keys: [flavor size]
v1 -> map[color:blue]
v2 -> map[color:blue flavor:strawberry size:big]`},
			{v1: []int{1}, v2: []int{1, 2}, expectedTrace: `
v1 does not contain v2[1]: "2"
v1 -> [1]
v2 -> [1 2]`},
			{v1: []string{"red", "green"}, v2: "blue", expectedTrace: `
v1 does not contain v2
v1 -> [red green]
v2 -> blue`},
			{v1: dict{"colors": dict{"color": "red"}}, v2: dict{"colors": dict{"color": "blue"}},
				expectedTrace: `
values are not equal
v1.colors.color -> red
v2.colors.color -> blue`},
			{v1: dict{"time": now}, v2: dict{"time": now.Add(time.Minute)}, opts: []ContainsOption{ParseTimes()},
				expectedTrace: `
values are not equal
v1.time -> 1987-02-10T06:30:15-05:00
v2.time -> 1987-02-10T06:31:15-05:00`,
			},
			{v1: dict{"time": now}, v2: dict{"time": now.Add(time.Minute)}, opts: []ContainsOption{AllowTimeDelta(time.Second * 30)},
				expectedTrace: `
delta of 1m0s exceeds 30s
v1.time -> 1987-02-10T06:30:15-05:00
v2.time -> 1987-02-10T06:31:15-05:00`,
			},
			{v1: dict{"time": now}, v2: dict{"time": nowCST}, opts: []ContainsOption{ParseTimes()},
				expectedTrace: `
time zone offsets don't match
v1.time -> 1987-02-10T06:30:15-05:00
v2.time -> 1987-02-10T05:30:15-06:00`,
			},
		}
		for _, test := range tests {
			t.Run("", func(t *testing.T) {
				var trace string
				Contains(test.v1, test.v2, append(test.opts, Trace(&trace))...)
				t.Log(trace)
				// strip off the leading new line.  Only there to make the test more readable
				assert.Equal(t, test.expectedTrace[1:], trace)
			})
		}
	})
}

func TestNormalize_proto(t *testing.T) {
	s := proto.Sample{
		Name:      "frank",
		IsEnabled: true,
	}

	v, err := Normalize(&s)
	require.NoError(t, err)
	assert.Equal(t, dict{"name": "frank", "active": true}, v)
}

func TestContainsMatch(t *testing.T) {
	w1 := Widget{
		Size:  1,
		Color: "red",
	}
	w2 := Widget{
		Size:  1,
		Color: "red",
	}

	m := ContainsMatch(w1, w2)
	assert.True(t, m.Matches)
	assert.Equal(t, dict{"size": float64(1), "color": "red"}, m.V1)
	assert.Equal(t, dict{"size": float64(1), "color": "red"}, m.V2)

	w1.Color = "redblue"
	m = ContainsMatch(w1, w2)
	assert.False(t, m.Matches)
	assert.Equal(t, dict{"size": float64(1), "color": "redblue"}, m.V1)
	assert.Equal(t, dict{"size": float64(1), "color": "red"}, m.V2)

	m = ContainsMatch(w1, w2, StringContains())
	assert.True(t, m.Matches)
}

func TestEquivalent(t *testing.T) {
	v1 := dict{"size": 1, "color": "big", "flavor": "mint"}
	v2 := Widget{
		Size:  1,
		Color: "big",
	}
	assert.True(t, Contains(v1, v2))

	var trace string
	assert.False(t, Equivalent(v1, v2, Trace(&trace)))

	assert.Equal(t, `v1 contains extra keys: [flavor]
v1 -> map[color:big flavor:mint size:1]
v2 -> map[color:big size:1]`, trace)

	// inverse should also be false
	assert.False(t, Equivalent(v2, v1))

	v1 = dict{"size": 1, "color": "big"}
	assert.True(t, Equivalent(v1, v2))
	assert.True(t, Equivalent(v2, v1))

	// StringContains should work too, ignoring the inverse comparison
	v1["color"] = "bigred"
	assert.False(t, Equivalent(v1, v2))
	assert.True(t, Equivalent(v1, v2, StringContains()))

	// EmptyValuesMatchAny should also work, like StringContains
	v2.Color = ""
	assert.False(t, Equivalent(v1, v2))
	assert.True(t, Equivalent(v1, v2, EmptyValuesMatchAny()))
}

func TestEquivalentMatch(t *testing.T) {
	w1 := Widget{
		Size:  1,
		Color: "red",
	}
	w2 := Widget{
		Size:  1,
		Color: "red",
	}

	m := EquivalentMatch(w1, w2)
	assert.True(t, m.Matches)
	assert.Equal(t, dict{"size": float64(1), "color": "red"}, m.V1)
	assert.Equal(t, dict{"size": float64(1), "color": "red"}, m.V2)

	w1.Color = "redblue"
	m = EquivalentMatch(w1, w2)
	assert.False(t, m.Matches)
	assert.Equal(t, dict{"size": float64(1), "color": "redblue"}, m.V1)
	assert.Equal(t, dict{"size": float64(1), "color": "red"}, m.V2)

	m = EquivalentMatch(w1, w2, StringContains())
	assert.True(t, m.Matches)
}

type dict = map[string]interface{}

func TestConflicts(t *testing.T) {
	tests := []struct {
		m1, m2   dict
		expected bool
	}{
		{
			dict{"color": "red"},
			dict{"temp": "hot"},
			false,
		},
		{
			dict{"color": "red"},
			dict{"color": "blue"},
			true,
		},
		{
			dict{"color": "red"},
			dict{"temp": "hot", "color": "red"},
			false,
		},
		{
			dict{
				"labels": dict{"region": "west"},
			},
			dict{
				"temp":   "hot",
				"labels": dict{"region": "west"},
			},
			false,
		},
		{
			dict{
				"labels": dict{"region": "east"},
			},
			dict{
				"temp":   "hot",
				"labels": dict{"region": "west"},
			},
			true,
		},
		{
			dict{
				"tags": []string{"green", "red"},
			},
			dict{
				"tags": []string{"orange", "black"},
			},
			false,
		},
		{
			dict{
				"tags": []string{"green", "red"},
			},
			dict{
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
	m1 := dict{}
	m1["matches"] = bigNestedMaps("color", 5)
	m1["notmatches"] = bigNestedMaps("weather", 5)
	s1 := []interface{}{}
	for i := 0; i < 100; i++ {
		s1 = append(s1, bigNestedMaps(fmt.Sprintf("food%v", i), 3))
	}
	m1["slice"] = s1
	m2 := dict{}
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
		Merge(dict{
			"colors": dict{
				"warm": "orange",
			},
			"numbers": dict{
				"odd":  3,
				"even": 4,
			},
		}, dict{
			"hot": "roof",
			"colors": dict{
				"warm": "red",
				"cool": "blue",
			},
			"flavors": dict{
				"sweet": "chocolate",
			},
		},
		)
	}
}

func toMap(s string) (out dict) {
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
		{in: dict{"red": "green"}, out: dict{"red": "green"}},
		{in: []interface{}{"red", 4}, out: []interface{}{"red", float64(4)}},
		{in: []string{"red", "green"}, out: []interface{}{"red", "green"}},
		// hits the marshaling path
		{in: &Widget{5, "red"}, out: dict{"size": float64(5), "color": "red"}},
		// marshaling might occur deep
		{in: dict{"widget": &Widget{5, "red"}}, out: dict{"widget": dict{"size": float64(5), "color": "red"}}},
		{
			name: "marshaller",
			in:   json.RawMessage(`{"color":"blue"}`),
			out:  dict{"color": "blue"},
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
		{dict{"color": "red"}, "red", "color"},
		{dict{"tags": []string{"red", "green"}}, "green", "tags[1]"},
		{dict{"tags": []string{"red", "green"}}, "red", "tags[0]"},
		{dict{"resource": dict{"tags": []string{"red", "green"}}}, "red", "resource.tags[0]"},
		{dict{"resource": dict{"tags": []string{"red", "green"}}}, []string{"red", "green"}, "resource.tags"},
		{dict{"resource": dict{"tags": []string{"red", "green"}}}, dict{"tags": []string{"red", "green"}}, "resource"},
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
		{dict{"tags": []string{"red", "green"}}, "tags[2]", "Index out of bounds at tags[2] (len = 2)", IndexOutOfBoundsError},
		{[]string{"red", "green"}, "[2]", "Index out of bounds at [2] (len = 2)", IndexOutOfBoundsError},
		{dict{"tags": "red"}, "tags[2]", "tags is not a slice", PathNotSliceError},
		{dict{"tags": "red"}, "[2]", "v is not a slice", PathNotSliceError},
		{[]string{"red", "green"}, "tags[2]", "v is not a map", PathNotMapError},
		{dict{"tags": "red"}, "color", "color not found", PathNotFoundError},
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
	type St struct {
		a, b string
		i    int
		ar   []string
	}
	var s St
	emptyTests := []interface{}{
		nil, "", "  ", dict{},
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
		s,
		St{},

		make(chan string),
		holder{},
	}
	for _, v := range emptyTests {
		assert.True(t, Empty(v), "v = %#v", v)
	}
	notEmptyTests := []interface{}{
		5, float64(5), true,
		"asdf", " asdf ", dict{"color": "red"},
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
	in := dict{
		"color": "red",
		"size":  5,
		"tags": []interface{}{
			"blue",
			false,
			nil,
		},
		"labels": dict{
			"region": "east",
		},
	}
	transformer := func(in interface{}) (interface{}, error) {
		if s, ok := in.(string); ok {
			return s + "s", nil
		}
		return in, nil
	}
	expected := dict{
		"color": "reds",
		"size":  float64(5),
		"tags": []interface{}{
			"blues",
			false,
			nil,
		},
		"labels": dict{
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
		case dict:
			delete(t, "labels")
		case []interface{}:
			in = append(t, "dogs")
		}
		return in, nil
	}

	expected = dict{
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
	s := dict{
		"Map": map[string]string{
			"color": "blue",
		},
		"Slice": []string{"green", "yellow"},
	}
	for i := 0; i < b.N; i++ {
		Normalize(s)
	}
}
