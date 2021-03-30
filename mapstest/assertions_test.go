package mapstest

import (
	"fmt"
	maps "github.com/ansel1/vespucci/v4"
	"github.com/stretchr/testify/assert"
	"testing"
)

type mockTestingT struct {
	msg       string
	failed    bool
	failedNow bool
}

func (m *mockTestingT) Logf(_ string, _ ...interface{}) {

}

func (m *mockTestingT) Errorf(s string, a ...interface{}) {
	m.msg = fmt.Sprintf(s, a...)
	m.failed = true
}

func (m *mockTestingT) FailNow() {
	m.failedNow = true
}

type dict = map[string]interface{}

func TestAssertionsContains(t *testing.T) {

	tests := []struct {
		v1, v2   interface{}
		contains bool
		equiv    bool
		err      bool
		opts     []interface{}
	}{
		{
			v1:       "red",
			v2:       "red",
			contains: true,
			equiv:    true,
		},
		{
			v1:       "red",
			v2:       "blue",
			contains: false,
			equiv:    false,
		},
		{
			v1:       "red",
			v2:       "",
			contains: true,
			equiv:    true,
		},
		{
			v1:       "red",
			v2:       "blue",
			contains: false,
			equiv:    false,
			opts:     []interface{}{Strict},
		},
		{
			v1:       "redblue",
			v2:       "blue",
			contains: true,
			equiv:    true,
			opts:     []interface{}{maps.StringContains()},
		},
		{
			v1:       dict{"color": "red", "size": 1},
			v2:       dict{"color": "red"},
			contains: true,
			equiv:    false,
		},
		{
			v1:  "string",
			v2:  make(chan bool),
			err: true,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			t.Logf("v1: %+v", test.v1)
			t.Logf("v2: %+v", test.v2)

			type assertFunc func(TestingT, interface{}, interface{}, ...interface{}) bool
			type requireFunc func(TestingT, interface{}, interface{}, ...interface{})

			af := func(fn assertFunc, expectSuccess bool) {
				mt := mockTestingT{}
				b := fn(&mt, test.v1, test.v2, test.opts...)
				t.Logf("msg: " + mt.msg)
				assert.Equal(t, expectSuccess, b)
				if expectSuccess {
					assert.False(t, mt.failed)
					assert.False(t, mt.failedNow)
				} else {
					assert.True(t, mt.failed)
					assert.False(t, mt.failedNow)
				}
			}

			rf := func(fn requireFunc, expectSuccess bool) {
				mt := mockTestingT{}
				fn(&mt, test.v1, test.v2, test.opts...)
				t.Logf("msg: " + mt.msg)
				if expectSuccess {
					assert.False(t, mt.failed)
					assert.False(t, mt.failedNow)
				} else {
					assert.True(t, mt.failed)
					assert.True(t, mt.failedNow)
				}
			}

			af(AssertContains, test.contains && !test.err)
			af(AssertNotContains, !test.contains && !test.err)
			rf(RequireContains, test.contains && !test.err)
			rf(RequireNotContains, !test.contains && !test.err)

			af(AssertEquivalent, test.equiv && !test.err)
			af(AssertNotEquivalent, !test.equiv && !test.err)
			rf(RequireEquivalent, test.equiv && !test.err)
			rf(RequireNotEquivalent, !test.equiv && !test.err)

		})
	}
}
