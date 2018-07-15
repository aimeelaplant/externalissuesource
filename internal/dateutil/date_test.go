package dateutil

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

func TestCompareMonths(t *testing.T) {
	dateOne, err := time.Parse("January 6 2006", "January 6 2006")
	if err != nil {
		t.Errorf(err.Error())
	}
	dateTwo, err := time.Parse("January 6 2006", "December 6 2005")
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Equal(t, 1, CompareMonths(dateOne, dateTwo))
	assert.Equal(t, -1, CompareMonths(dateTwo, dateOne))

	dateOne, err = time.Parse("January 6 2006", "January 6 2006")
	if err != nil {
		t.Errorf(err.Error())
	}
	dateTwo, err = time.Parse("January 6 2006", "December 6 2006")
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Equal(t, -11, CompareMonths(dateOne, dateTwo))
	assert.Equal(t, 11, CompareMonths(dateTwo, dateOne))
}