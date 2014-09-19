package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRingBuffer(t *testing.T) {
	b := make([]byte, 10)
	buf := NewRingBuffer(5)

	n, _ := buf.Write([]byte{1, 2, 3})
	assert.Equal(t, 3, n)
	n, _ = buf.Read(b)
	assert.Equal(t, []byte{1, 2, 3}, b[:n])

	n, _ = buf.Write([]byte{4, 5, 6})
	assert.Equal(t, 3, n)
	n, _ = buf.Read(b)
	assert.Equal(t, []byte{2, 3, 4, 5, 6}, b[:n])

	n, _ = buf.Write([]byte{7, 8, 9, 10})
	assert.Equal(t, 4, n)
	n, _ = buf.Read(b)
	assert.Equal(t, []byte{6, 7, 8, 9, 10}, b[:n])

	b = b[:2]
	n, _ = buf.Read(b)
	assert.Equal(t, []byte{9, 10}, b[:n])

	buf.Clear()
	n, _ = buf.Read(b)
	assert.Equal(t, []byte{}, b[:n])
}
