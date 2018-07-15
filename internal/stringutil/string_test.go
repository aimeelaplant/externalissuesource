package stringutil

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestRandString(t *testing.T) {
	s := RandString(26)
	assert.NotEmpty(t, s)
	assert.Len(t, s, 26)

	s = RandString(32)
	assert.NotEmpty(t, s)
	assert.Len(t, s, 32)
}