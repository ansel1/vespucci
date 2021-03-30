package mapstest

import (
	"fmt"
	maps "github.com/ansel1/vespucci/v4"
	"github.com/davecgh/go-spew/spew"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type strictMarker int

// Strict is an option that can be passed to the Contains and Equivalent assertions.  It
// disables the default ContainsOptions.
const Strict strictMarker = 0

// AssertContains returns true if maps.Contains(v1, v2).  The following
// ContainsOptions are automatically applied:
//
// - maps.EmptyMapValuesMatchAny
// - maps.IgnoreTimeZones(true)
// - maps.ParseTimes
//
// These default options can be suppressed by passing Strict in the options:
//
//     AssertContains(t, v1, v2, Strict)
//
// optsMsgAndArgs can contain a string msg and a series of args, which
// will be formatted into the assertion failure message.
//
// optsMsgAndArgs may also contain additional ContainOptions, which will be extracted
// and applied to the Contains() function.
func AssertContains(t TestingT, v1, v2 interface{}, optsMsgAndArgs ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	opts, optsMsgAndArgs := splitOptions(optsMsgAndArgs)
	match := maps.ContainsMatch(v1, v2, opts...)
	if !assert.NoError(t, match.Error, match.Message) {
		return false
	}

	if !match.Matches {
		nv1, err := maps.Normalize(v1)
		if assert.NoError(t, err, "error normalizing v1") {
			v1 = nv1
		}
		nv2, err := maps.Normalize(v2)
		if assert.NoError(t, err, "error normalizing v2") {
			v2 = nv2
		}
		diff := containsDiff(v1, v2)
		return assert.Fail(t, fmt.Sprintf("v1 does not contain v2: \n"+
			"%s%s", match.Message, diff), optsMsgAndArgs...)
	}

	return true
}

// AssertNotContains is the inverse of AssertContains
func AssertNotContains(t TestingT, v1, v2 interface{}, optsMsgAndArgs ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	opts, optsMsgAndArgs := splitOptions(optsMsgAndArgs)
	match := maps.ContainsMatch(v1, v2, opts...)
	if !assert.NoError(t, match.Error, match.Message) {
		return false
	}

	if match.Matches {
		nv1, err := maps.Normalize(v1)
		if assert.NoError(t, err, "error normalizing v1") {
			v1 = nv1
		}
		nv2, err := maps.Normalize(v2)
		if assert.NoError(t, err, "error normalizing v2") {
			v2 = nv2
		}
		return assert.Fail(t, fmt.Sprintf("v1 should not contain v2: \n"+
			"v1: %+v\n"+
			"v2: %+v", v1, v2), optsMsgAndArgs...)
	}

	return true
}

// AssertEquivalent returns true if maps.Equivalent(v1, v2).  The following
// ContainsOptions are automatically applied:
//
// - maps.EmptyMapValuesMatchAny
// - maps.IgnoreTimeZones(true)
// - maps.ParseTimes
//
// optsMsgAndArgs can contain a string msg and a series of args, which
// will be formatted into the assertion failure message.
//
// optsMsgAndArgs may also contain additional ContainOptions, which will be extracted
// and applied to the Equivalent() function.
func AssertEquivalent(t TestingT, v1, v2 interface{}, optsMsgAndArgs ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	opts, optsMsgAndArgs := splitOptions(optsMsgAndArgs)
	match := maps.EquivalentMatch(v1, v2, opts...)
	if !assert.NoError(t, match.Error, match.Message) {
		return false
	}

	if !match.Matches {
		nv1, err := maps.Normalize(v1)
		if assert.NoError(t, err, "error normalizing v1") {
			v1 = nv1
		}
		nv2, err := maps.Normalize(v2)
		if assert.NoError(t, err, "error normalizing v2") {
			v2 = nv2
		}
		return assert.Fail(t, fmt.Sprintf("v1 !≈ v2: \n"+
			"%s%s", match.Message, containsDiff(v1, v2)), optsMsgAndArgs...)
	}

	return true
}

// AssertNotEquivalent is the inverse of AssertEquivalent
func AssertNotEquivalent(t TestingT, v1, v2 interface{}, optsMsgAndArgs ...interface{}) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	opts, optsMsgAndArgs := splitOptions(optsMsgAndArgs)
	match := maps.EquivalentMatch(v1, v2, opts...)
	if !assert.NoError(t, match.Error, match.Message) {
		return false
	}

	if match.Matches {
		nv1, err := maps.Normalize(v1)
		if assert.NoError(t, err, "error normalizing v1") {
			v1 = nv1
		}
		nv2, err := maps.Normalize(v2)
		if assert.NoError(t, err, "error normalizing v2") {
			v2 = nv2
		}
		return assert.Fail(t, fmt.Sprintf("v1 should not ≈ v2: \n"+
			"v1: %+v\n"+
			"v2: %+v", v1, v2), optsMsgAndArgs...)
	}

	return true
}

// RequireContains is like AssertContains, but fails the test immediately.
func RequireContains(t TestingT, v1, v2 interface{}, optsMsgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !AssertContains(t, v1, v2, optsMsgAndArgs...) {
		t.FailNow()
	}
}

// RequireNotContains is like AssertNotContains, but fails the test immediately.
func RequireNotContains(t TestingT, v1, v2 interface{}, optsMsgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !AssertNotContains(t, v1, v2, optsMsgAndArgs...) {
		t.FailNow()
	}
}

// RequireEquivalent is like AssertEquivalent, but fails the test immediately.
func RequireEquivalent(t TestingT, v1, v2 interface{}, optsMsgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !AssertEquivalent(t, v1, v2, optsMsgAndArgs...) {
		t.FailNow()
	}
}

// RequireNotEquivalent is like AssertNotEquivalent, but fails the test immediately.
func RequireNotEquivalent(t TestingT, v1, v2 interface{}, optsMsgAndArgs ...interface{}) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	if !AssertNotEquivalent(t, v1, v2, optsMsgAndArgs...) {
		t.FailNow()
	}
}

var spewC = spew.ConfigState{
	Indent:                  " ",
	DisablePointerAddresses: true,
	DisableCapacities:       true,
	SortKeys:                true,
}

// containsDiff returns a diff of both values as long as both are of the same type and
// are a struct, map, slice or array. Otherwise it returns an empty string.
func containsDiff(v1 interface{}, v2 interface{}) string {

	e := spewC.Sdump(v1)
	a := spewC.Sdump(v2)

	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(e),
		B:        difflib.SplitLines(a),
		FromFile: "v1",
		FromDate: "",
		ToFile:   "v2",
		ToDate:   "",
		Context:  1,
	})

	return "\n\nDiff:\n" + diff
}

// removes any instances of DeepContainsOption from args, and uses them to create
// a deepContainsOptions.  Returns the initialized options, which will never be nil,
// and any remaining items in args.
func splitOptions(args []interface{}) (opts []maps.ContainsOption, msgAndArgs []interface{}) {
	msgAndArgs = args[:0]
	var strict bool

	for _, arg := range args {
		switch t := arg.(type) {
		case strictMarker:
			strict = true
		case maps.ContainsOption:
			opts = append(opts, t)
		default:
			msgAndArgs = append(msgAndArgs, arg)
		}
	}

	if !strict {
		opts = append(opts,
			maps.EmptyMapValuesMatchAny(),
			maps.IgnoreTimeZones(true),
			maps.ParseTimes(),
		)
	}

	return
}

type tHelper interface {
	Helper()
}

// TestingT is a subset of testing.TB
type TestingT = require.TestingT
