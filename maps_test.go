package maps

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"encoding/json"
	"sort"
)

func TestMerge(t *testing.T) {
	tests := []struct{
		src, dst, expected string
	}{
		{`{"hot":"weather"}`, `{"hard":"lemonade"}`, `{"hot":"weather","hard":"lemonade"}`},
		{`{"hot":"weather"}`, `{"hot":"roof"}`, `{"hot":"weather"}`},
		{`{"colors":{"color":"orange"}}`, `{"hot":"roof"}`, `{"hot":"roof","colors":{"color":"orange"}}`},
		{`{"colors":{"warm":"orange"}, "numbers":{"odd":3, "even":4}}`, `{"hot":"roof", "colors":{"warm":"red","cool":"blue"}, "flavors":{"sweet":"chocolate"}}`, `{"hot":"roof", "colors":{"warm":"orange","cool":"blue"}, "flavors":{"sweet":"chocolate"}, "numbers":{"odd":3, "even":4}}`},
		{`{"tags":["east","blue"]}`, `{"tags":["east","loud","big"]}`, `{"tags":["east", "loud", "big", "blue"]}`},
	}
	for _, test := range tests {
		dst := toMap(test.dst)
		Merge(dst, toMap(test.src), true)
		assert.Equal(t, toMap(test.expected), dst)
	}
}

func TestKeys(t *testing.T) {
	tests := []struct {
		m map[string]interface{}
		k []string
	}{
		{map[string]interface{}{"color":"blue","price":"high","weight":2}, []string{"color","price","weight"}},
	}
	for _, test := range tests {
		sort.Strings(test.k)
		out := Keys(test.m)
		sort.Strings(out)
		assert.Equal(t, test.k, out)
	}
}

func BenchmarkMerge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Merge(map[string]interface{}{
			"colors": map[string]interface{}{
				"warm":"orange",
			},
			"numbers": map[string]interface{}{
				"odd":3,
				"even":4,
			},
		},map[string]interface{}{
			"hot": "roof",
			"colors": map[string]interface{}{
				"warm":"red",
				"cool":"blue",
			},
			"flavors": map[string]interface{}{
				"sweet":"chocolate",
			},
		}, false,
		)
	}
}

func toMap(s string) (out map[string]interface{}) {
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		panic(err)
	}
	return
}