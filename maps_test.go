package maps

import (
	"encoding/json"
	"fmt"
	"github.com/k0kubun/pp"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
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
	}
	for _, test := range literalMapsTests {
		r := Merge(test.m1, test.m2)
		assert.Equal(t, test.m3, r)
	}
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
	tests := []struct {
		v1, v2   interface{}
		expected bool
	}{
		{"red", "red", true},
		{"red", "green", false},
		{
			[]string{"big", "loud"},
			[]string{"smart", "loud"},
			false,
		},
		{
			map[string]interface{}{"resource": map[string]interface{}{"id": 1, "color": "red", "tags": []string{"big", "loud"}}, "environment": map[string]interface{}{"time": "night", "source": "east"}},
			map[string]interface{}{"resource": map[string]interface{}{"tags": []string{"smart", "loud"}}},
			false,
		},
		{
			map[string]interface{}{"color": "red"},
			map[string]interface{}{"color": "red"},
			true,
		},
		{
			map[string]interface{}{"color": "green"},
			map[string]interface{}{"color": "red"},
			false,
		},
		{
			map[string]interface{}{"color": "green", "flavor": "beef"},
			map[string]interface{}{"color": "green"},
			true,
		},
		{
			map[string]interface{}{"color": "green", "tags": []string{"beef", "hot"}},
			map[string]interface{}{"color": "green", "tags": []string{"hot"}},
			true,
		},
		{
			map[string]interface{}{"color": "green", "tags": []string{"beef", "hot"}},
			map[string]interface{}{"color": "green", "tags": []string{"cool"}},
			false,
		},
		{
			map[string]interface{}{
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
			map[string]interface{}{
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
			true,
		},
		{
			map[string]interface{}{
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
			map[string]interface{}{
				"resource": map[string]interface{}{
					"size": 7,
				},
			},
			false,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, Contains(test.v1, test.v2), pp.Sprintln("m1", test.v1, "m2", test.v2))
	}
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
