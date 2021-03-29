package main

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestUrlPolish(t *testing.T) {
	out := "category/<id>"

	in1 := "category/123"
	in2 := "category/123-456"

	assert.Equal(t, urlPolish(in1), out)
	assert.Equal(t, urlPolish(in2), out)

	in3 := "category/name"

	assert.Equal(t, urlPolish(in3), in3)
}