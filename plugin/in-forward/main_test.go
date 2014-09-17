package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseTime(t *testing.T) {
	epoch := time.Now().Unix()
	now := time.Unix(epoch, 0)

	tt, err := parseTime(int64(epoch))
	assert.Equal(t, nil, err)
	assert.Equal(t, now, tt)

	tt, err = parseTime(uint64(epoch))
	assert.Equal(t, nil, err)
	assert.Equal(t, now, tt)

	tt, err = parseTime(float64(epoch))
	assert.Equal(t, nil, err)
	assert.Equal(t, now, tt)

	tt, err = parseTime(uint32(epoch))
	assert.Equal(t, nil, err)
	assert.Equal(t, now, tt)
}
