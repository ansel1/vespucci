package maps

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ansel1/merry"
	"github.com/ansel1/vespucci/v4/proto"
	"github.com/davecgh/go-spew/spew"
	"github.com/k0kubun/pp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"sort"
	"strconv"
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
			v1:       "red",
			v2:       "green",
			options:  []ContainsOption{StringContains()},
			expected: false,
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
			name:     "parsetimes",
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
			v1:       t1.In(time.FixedZone("test", -3*60*60)),
			v2:       t1.UTC(),
			options:  []ContainsOption{},
			expected: false,
		},
		{
			name:     "timezones parsed",
			v1:       t1.In(time.FixedZone("test", -3*60*60)),
			v2:       t1.UTC(),
			options:  []ContainsOption{ParseTimes()},
			expected: false,
		},
		{
			name:     "not ignore timezones",
			v1:       t1.In(time.FixedZone("test", -3*60*60)),
			v2:       t1.UTC(),
			options:  []ContainsOption{IgnoreTimeZones(false)},
			expected: false,
		},
		{
			name:     "ignore timezones",
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

	spewConf := spew.NewDefaultConfig()
	spewConf.SortKeys = true
	spewConf.SpewKeys = true
	spewConf.DisablePointerMethods = true

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v1Str := spewConf.Sprintf("%#+v", test.v1)
			v2Str := spewConf.Sprintf("%#+v", test.v2)
			assert.Equal(t, test.expected, Contains(test.v1, test.v2, test.options...), pp.Sprintln("m1", test.v1, "m2", test.v2))
			// make sure contains didn't modify the input values at all
			assert.Equal(t, v1Str, spewConf.Sprintf("%#+v", test.v1))
			assert.Equal(t, v2Str, spewConf.Sprintf("%#+v", test.v2))
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
v1 -> "red"
v2 -> "blue"`},
			{v1: "red", v2: 1, expectedTrace: `
values are not equal
v1 -> "red"
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
v1 -> "red"
v2 -> "blue"`},
			{v1: dict{"color": "red"}, v2: 1, expectedTrace: `
values are not equal
v1 -> map[string]interface {}{"color":"red"}
v2 -> 1`},
			{v1: dict{"color": "red"}, v2: dict{"color": "blue"}, expectedTrace: `
values are not equal
v1.color -> "red"
v2.color -> "blue"`},
			{v1: dict{"color": dict{"height": "tall"}}, v2: dict{"color": dict{"height": "short"}}, expectedTrace: `
values are not equal
v1.color.height -> "tall"
v2.color.height -> "short"`},
			{v1: dict{"color": "blue"}, v2: dict{"color": "blue", "size": "big", "flavor": "strawberry"}, expectedTrace: `
v2 contains extra keys: [flavor size]
v1 -> map[string]interface {}{"color":"blue"}
v2 -> map[string]interface {}{"color":"blue", "flavor":"strawberry", "size":"big"}`},
			{v1: []int{1}, v2: []int{1, 2}, expectedTrace: `
v1 does not contain v2[1]: "2"
v1 -> []interface {}{1}
v2 -> []interface {}{1, 2}`},
			{v1: []string{"red", "green"}, v2: "blue", expectedTrace: `
v1 does not contain v2
v1 -> []interface {}{"red", "green"}
v2 -> "blue"`},
			{v1: dict{"colors": dict{"color": "red"}}, v2: dict{"colors": dict{"color": "blue"}},
				expectedTrace: `
values are not equal
v1.colors.color -> "red"
v2.colors.color -> "blue"`},
			{v1: dict{"time": now}, v2: dict{"time": now.Add(time.Minute)}, opts: []ContainsOption{ParseTimes()},
				expectedTrace: `
values are not equal
v1.time -> "1987-02-10 06:30:15 -0500 EST"
v2.time -> "1987-02-10 06:31:15 -0500 EST"`,
			},
			{v1: dict{"time": now}, v2: dict{"time": now.Add(time.Minute)}, opts: []ContainsOption{AllowTimeDelta(time.Second * 30)},
				expectedTrace: `
delta of 1m0s exceeds 30s
v1.time -> "1987-02-10 06:30:15 -0500 EST"
v2.time -> "1987-02-10 06:31:15 -0500 EST"`,
			},
			{v1: dict{"time": now}, v2: dict{"time": nowCST}, opts: []ContainsOption{ParseTimes()},
				expectedTrace: `
time zone offsets don't match
v1.time -> "1987-02-10 06:30:15 -0500 EST"
v2.time -> "1987-02-10 05:30:15 -0600 CST"`,
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
	assert.Empty(t, m.Message)
	assert.Nil(t, m.V1)
	assert.Nil(t, m.V2)
	assert.Nil(t, m.Error)
	assert.Empty(t, m.Path)

	w1.Color = "redblue"
	m = ContainsMatch(w1, w2)
	assert.False(t, m.Matches)
	assert.NotEmpty(t, m.Message)
	assert.Equal(t, "redblue", m.V1)
	assert.Equal(t, "red", m.V2)
	assert.Nil(t, m.Error)
	assert.Equal(t, "color", m.Path)

	m = ContainsMatch(w1, w2, StringContains())
	assert.True(t, m.Matches)

	// try something that will cause a marshaling error, like a channel value
	m = ContainsMatch(w1, dict{"size": 1, "color": make(chan string)})
	assert.Error(t, m.Error)
	assert.Contains(t, m.Error.Error(), "json: unsupported type")
	assert.False(t, m.Matches)
	assert.NotEmpty(t, m.Message)
	assert.Contains(t, m.Message, "err normalizing v2")
	assert.Equal(t, "color", m.Path)
	assert.Equal(t, "redblue", m.V1)
	_, ok := m.V2.(chan string)
	assert.True(t, ok, "should have been a channel, was %T", m.V2)
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
v1 -> map[string]interface {}{"color":"big", "flavor":"mint", "size":1}
v2 -> map[string]interface {}{"color":"big", "size":1}`, trace)

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

	// slice values must be slices on both sides
	assert.True(t, Contains([]interface{}{"blue", "red", "green"}, "red"))
	assert.False(t, Equivalent([]interface{}{"blue", "red", "green"}, "red"))

	// slice values must contain the same values, but order doesn't matter
	assert.False(t, Equivalent([]interface{}{"blue", "red", "green"}, []interface{}{"red", "green", "orange"}))
	assert.False(t, Equivalent([]interface{}{"blue", "red", "green"}, []interface{}{"red", "green", "blue", "orange"}))

	// StringContains should still work
	assert.True(t, Equivalent([]interface{}{"bluegreen", "red", "green"}, []interface{}{"red", "green", "blue"}, StringContains()))
	assert.False(t, Equivalent([]interface{}{"blue", "red", "green"}, []interface{}{"red", "green", "bluegreen"}, StringContains()))

	// nils with EmptyValuesMatchAny work like wildcards
	assert.True(t, Equivalent([]interface{}{"blue", "red", "green", "black"}, []interface{}{"red", "red", "green", ""}, EmptyMapValuesMatchAny()))

	assert.True(t, Equivalent([]interface{}{"blue", "red", "green", "green"}, []interface{}{"red", "red", "green", "blue"}))
	assert.False(t, Equivalent([]interface{}{"blue", "red", "green", "black"}, []interface{}{"red", "red", "green", "blue"}))
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
	assert.Empty(t, m.Message)

	w1.Color = "redblue"
	m = EquivalentMatch(w1, w2)
	assert.False(t, m.Matches)
	assert.NotEmpty(t, m.Message)

	m = EquivalentMatch(w1, w2, StringContains())
	assert.True(t, m.Matches)

	// try something that will cause a marshaling error, like a channel value
	m = EquivalentMatch(w1, dict{"size": 1, "color": make(chan string)})
	assert.Error(t, m.Error)
	assert.Contains(t, m.Error.Error(), "json: unsupported type")
	assert.False(t, m.Matches)
	assert.NotEmpty(t, m.Message)
	assert.Contains(t, m.Message, "err normalizing v2")
	assert.Equal(t, "color", m.Path)
	assert.Equal(t, "redblue", m.V1)
	_, ok := m.V2.(chan string)
	assert.True(t, ok, "should have been a channel, was %T", m.V2)
}

type dict = map[string]any

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

	b.Run("withCopy", func(b *testing.B) {
		// pp.Println("m1", m1)
		for i := 0; i < b.N; i++ {
			Merge(m1, m2)
		}
	})

	b.Run("noCopy", func(b *testing.B) {
		// pp.Println("m1", m1)
		for i := 0; i < b.N; i++ {
			Merge(m1, m2, Copy(false))
		}
	})
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

// specialTime marshals to a time string.
type specialTime struct {
	t time.Time
}

func (s *specialTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.t)
}

func TestNormalize(t *testing.T) {
	t1 := time.Date(1990, 11, 23, 2, 2, 2, 2, time.FixedZone("testzone", -3*60*60))

	n1, err := Normalize(t1)
	require.NoError(t, err)

	n2, err := Normalize(t1.UTC())
	require.NoError(t, err)

	m1, err := time.Parse(time.RFC3339Nano, n1.(string))
	require.NoError(t, err)
	m2, err := time.Parse(time.RFC3339Nano, n2.(string))
	require.NoError(t, err)

	assert.Zero(t, m1.Sub(m2))

	tests := []struct {
		name    string
		in, out interface{}
		opts    []NormalizeOption
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
		// hits the marshaling currentPath
		{in: &Widget{Size: 5, Color: "red"}, out: dict{"size": float64(5), "color": "red"}},
		// marshaling might occur deep
		{in: dict{"widget": &Widget{Size: 5, Color: "red"}}, out: dict{"widget": dict{"size": float64(5), "color": "red"}}},
		{
			name: "marshaller",
			in:   json.RawMessage(`{"color":"blue"}`),
			out:  dict{"color": "blue"},
			opts: []NormalizeOption{Marshal(true)},
		},
		{in: t1, out: "1990-11-23T02:02:02.000000002-03:00"},
		{in: t1.UTC(), out: "1990-11-23T05:02:02.000000002Z"},
		{in: t1, out: t1, opts: []NormalizeOption{NormalizeTime(true)}},
		{in: &specialTime{t: t1.UTC()}, out: t1.UTC(), opts: []NormalizeOption{NormalizeTime(true)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out interface{}
			var err error
			out, err = Normalize(test.in, test.opts...)

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
		require.NoError(t, err, "v = %#v, currentPath = %v", test.v, test.path)
		require.Equal(t, test.out, result, "v = %#v, currentPath = %v", test.v, test.path)
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
		assert.EqualError(t, err, test.msg, "v = %#v, currentPath = %v", test.v, test.path)
		assert.True(t, merry.Is(err, test.kind), "Wrong type of error.  Expected %v, was %v", test.kind, err)
	}
}

type holder struct {
	i interface{}
}

func TestEmpty(t *testing.T) {
	var num int
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
		time.Time{},
		(*int)(nil),
		num,
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
		time.Now(),
		&time.Time{},
		&num,
	}
	for _, v := range notEmptyTests {
		assert.False(t, Empty(v), "v = %#v", v)
	}
}

func BenchmarkEmpty(b *testing.B) {
	var w Widget
	b.Run("struct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Empty(w)
		}
	})

	b.Run("largeValue", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Empty(largeTestVal1)
		}
	})

	b.Run("primitive", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Empty(5)
		}
	})
}

func TestInternalNormalize(t *testing.T) {
	tm := time.Now()
	opts := NormalizeOptions{
		Copy:    false,
		Marshal: false,
		Deep:    true,
	}
	v, err := normalize(tm, &opts)
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

const largeTestVal1 string = `
{
	"principal": {
		"acct": "kylo:57b0b351-68f3-424c-be14-68a4a39d1255:admin:accounts:57b0b351-68f3-424c-be14-68a4a39d1255",
		"app": "ncryptify:gemalto:admin:apps:kylo",
		"aud": "3ee1f89e-6340-4123-91d7-a39eee330586",
		"cnf": null,
		"cust": {
		"groups": [
		  "CCKM Users"
		],
		"sid": "b0951adf-4521-40a9-8f6d-493bbb52af99",
		"zone_id": "00000000-0000-0000-0000-000000000000"
		},
		"dev_acct": "ncryptify:gemalto:admin:accounts:gemalto",
		"exp": 1688582702,
		"given_name": "",
		"iat": 1688582102,
		"ident": "ncryptify:gemalto:admin:identities:bob-bob",
		"iss": "kylo",
		"jti": "560bb8a3-42e7-46c5-9252-2c0587cf8ab2",
		"name": "",
		"nickname": "",
		"preferred_username": "bob",
		"sub": "bob",
		"sub_acct": "kylo:57b0b351-68f3-424c-be14-68a4a39d1255:admin:accounts:57b0b351-68f3-424c-be14-68a4a39d1255",
		"user": "ncryptify:gemalto:admin:users:bob"
	},
	"resource": {
		"id": "142514aecaff4329876579935829a052fcaf7753343843df833b2bfae72f2b36",
		"uri": "kylo:kylo:vault:secrets:ks-142514aecaff4329876579935829a052fcaf7753343843df833b2bfae72f2b36-v0",
		"account": "kylo:kylo:admin:accounts:kylo",
		"application": "ncryptify:gemalto:admin:apps:kylo",
		"devAccount": "ncryptify:gemalto:admin:accounts:gemalto",
		"createdAt": "2023-07-12T14:14:27.907976Z",
		"name": "ks-142514aecaff4329876579935829a052fcaf7753343843df833b2bfae72f2b36",
		"updatedAt": "2023-07-12T14:14:27.907976Z",
		"activationDate": "2023-07-12T14:14:27.890215Z",
		"state": "Active",
		"meta": {
			"description": "connection manager credential",
			"service_name": "connectionmgmt",
			"resource_type": "connections",
			"connection_uri": "kylo:kylo:connectionmgmt:connections:gcp-connection2-fc9d0a07-cf98-4d97-bce6-99a8ef9baf81"
		},
		"objectType": "Opaque Object",
		"sha1Fingerprint": "5725a93e4fa07b9d",
		"sha256Fingerprint": "691e82c107ba039ae5ed6b6863e2044ab8178a0fb40eabcb746e53f454d3bab7",
		"defaultIV": "3fa759e928dda71f84ffd631b2594d54",
		"version": 0,
		"algorithm": "OPAQUE",
		"unexportable": false,
		"undeletable": false,
		"neverExported": true,
		"neverExportable": false,
		"emptyMaterial": false,
		"uuid": "a7594090-12b0-40eb-b9ab-92d6f5f78fab"
	},
	"environment": {
		"id": "142514aecaff4329876579935829a052fcaf7753343843df833b2bfae72f2b36",
		"uri": "kylo:kylo:vault:secrets:ks-142514aecaff4329876579935829a052fcaf7753343843df833b2bfae72f2b36-v0",
		"account": "kylo:kylo:admin:accounts:kylo",
		"application": "ncryptify:gemalto:admin:apps:kylo",
		"devAccount": "ncryptify:gemalto:admin:accounts:gemalto",
		"createdAt": "2023-07-12T14:14:27.907976Z",
		"name": "ks-142514aecaff4329876579935829a052fcaf7753343843df833b2bfae72f2b36",
		"updatedAt": "2023-07-12T14:14:27.907976Z",
		"activationDate": "2023-07-12T14:14:27.890215Z",
		"state": "Active",
		"meta": {
			"description": "connection manager credential",
			"service_name": "connectionmgmt",
			"resource_type": "connections",
			"connection_uri": "kylo:kylo:connectionmgmt:connections:gcp-connection2-fc9d0a07-cf98-4d97-bce6-99a8ef9baf81"
		},
		"objectType": "Opaque Object",
		"sha1Fingerprint": "5725a93e4fa07b9d",
		"sha256Fingerprint": "691e82c107ba039ae5ed6b6863e2044ab8178a0fb40eabcb746e53f454d3bab7",
		"defaultIV": "3fa759e928dda71f84ffd631b2594d54",
		"version": 0,
		"algorithm": "OPAQUE",
		"unexportable": false,
		"undeletable": false,
		"neverExported": true,
		"neverExportable": false,
		"emptyMaterial": false,
		"uuid": "a7594090-12b0-40eb-b9ab-92d6f5f78fab",
		"obligations":{
			"blue": {
				"details": {
					"color": "blue"
				}
			}
		}
	}
}
`

func BenchmarkGet(b *testing.B) {
	// factor out the time to normalize
	n1, err := Normalize(json.RawMessage(largeTestVal1))
	require.NoError(b, err)

	// make sure it works as expected
	get, err := Get(n1, "environment.obligations.blue.details.color")
	require.NoError(b, err)
	require.Equal(b, "blue", get)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Get(n1, "environment.obligations.blue.details.color")

	}
}

func BenchmarkContains(b *testing.B) {
	// factor out the time to normalize
	n1, err := Normalize(json.RawMessage(largeTestVal1))
	require.NoError(b, err)

	matchingValue, err := Normalize(json.RawMessage(`
{
	"principal": {
		"cust": {
			"groups": ["CCKM Users"]
		}
	}
}	
	`))
	require.NoError(b, err)

	notMatchingValue, err := Normalize(json.RawMessage(`
{
	"principal": {
		"cust": {
			"groups": ["blue"]
		}
	}
}	
	`))
	require.NoError(b, err)

	b.Run("containsMismatchWithTrace", func(b *testing.B) {
		var traceMsg string

		for i := 0; i < b.N; i++ {
			Contains(n1, notMatchingValue, Trace(&traceMsg))
		}
	})

	b.Run("containsMismatchNoTrace", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Contains(n1, notMatchingValue)
		}
	})

	b.Run("containsMatching", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Contains(n1, matchingValue)
		}
	})

	b.Run("containsMatchMismatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ContainsMatch(n1, notMatchingValue)
		}
	})

	b.Run("containsMatchMatching", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ContainsMatch(n1, matchingValue)
		}
	})

}

func BenchmarkEquivalent(b *testing.B) {
	// factor out the time to normalize
	n1, err := Normalize(json.RawMessage(largeTestVal1))
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		Equivalent(n1, n1)
	}
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

func BenchmarkEquivalentSlices(b *testing.B) {
	// the toughest slice match is a large slice with lots of duplicates
	v1 := make([]string, 300)
	for i := 0; i < 100; i++ {
		v1[i] = strconv.Itoa(i)
	}
	copy(v1[100:], v1[0:100])
	copy(v1[200:], v1[100:200])

	rand.Shuffle(300, func(i, j int) {
		v1[i], v1[j] = v1[j], v1[i]
	})

	v2 := make([]string, 300)
	copy(v2, v1)
	rand.Shuffle(300, func(i, j int) {
		v2[i], v2[j] = v2[j], v2[i]
	})

	b.Run("large slices", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			m := EquivalentMatch(v1, v2)
			if !m.Matches {
				b.Fatal("the slices weren't equivalent: ", m.Message)
			}
		}
	})

	v1 = make([]string, 60)
	v2 = make([]string, 60)
	for i := 0; i < 30; i++ {
		v1[i] = strconv.Itoa(i)
	}
	copy(v1[30:], v1[0:30])
	copy(v2, v1)

	rand.Shuffle(60, func(i, j int) {
		v1[i], v1[j] = v1[j], v1[i]
	})
	rand.Shuffle(60, func(i, j int) {
		v2[i], v2[j] = v2[j], v2[i]
	})

	b.Run("short slices", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			m := EquivalentMatch(v1, v2)
			if !m.Matches {
				b.Fatal("the slices weren't equivalent: ", m.Message)
			}
		}
	})
}
