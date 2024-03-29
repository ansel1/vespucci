package mapstest

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// The tests in this file are very sensitive to source line numbers changing, so I'm
// putting this in it's own file.  If you move or change anything in this
// file that shifts line numbers, it will probably break these tests.

func TestAssertContains_message(t *testing.T) {
	mt := mockTestingT{}

	b := AssertContains(&mt, "red", "blue", "sample %v", 1)

	require.False(t, b)
	assert.Contains(t, mt.msg, `
	Error:      	v1 does not contain v2: 
	            	values are not equal
	            	v1 -> "red"
	            	v2 -> "blue"
	            	
	            	Diff:
	            	--- v1
	            	+++ v2
	            	@@ -1,2 +1,2 @@
	            	-(string) (len=3) "red"
	            	+(string) (len=4) "blue"
	            	 
	Messages:   	sample 1
`)
}

func TestAssertNotContains_message(t *testing.T) {
	mt := mockTestingT{}

	b := AssertNotContains(&mt, "red", "red", "sample %v", 1)

	require.False(t, b)
	assert.Contains(t, mt.msg, `
	Error:      	v1 should not contain v2: 
	            	v1: red
	            	v2: red
	Messages:   	sample 1
`)
}

func TestAssertEquivalent_message(t *testing.T) {
	mt := mockTestingT{}

	b := AssertEquivalent(&mt, "red", "blue", "sample %v", 1)

	require.False(t, b)
	assert.Contains(t, mt.msg, `
	Error:      	v1 !≈ v2: 
	            	values are not equal
	            	v1 -> "red"
	            	v2 -> "blue"
	            	
	            	Diff:
	            	--- v1
	            	+++ v2
	            	@@ -1,2 +1,2 @@
	            	-(string) (len=3) "red"
	            	+(string) (len=4) "blue"
	            	 
	Messages:   	sample 1
`)
}

func TestAssertNotEquivalent_message(t *testing.T) {
	mt := mockTestingT{}

	b := AssertNotEquivalent(&mt, "red", "red", "sample %v", 1)

	require.False(t, b)
	assert.Contains(t, mt.msg, `
	Error:      	v1 should not ≈ v2: 
	            	v1: red
	            	v2: red
	Messages:   	sample 1
`)
}
