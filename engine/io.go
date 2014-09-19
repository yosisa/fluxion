package engine

import "errors"

type RingBuffer struct {
	buf []byte
	pos int
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{buf: make([]byte, 0, size)}
}

func (w *RingBuffer) Write(b []byte) (int, error) {
	if cap(w.buf) == 0 {
		return 0, errors.New("Empty buffer")
	}

	n := len(b)
	if s := len(w.buf); s < cap(w.buf) {
		w.buf = w.buf[:min(s+n, cap(w.buf))]
	}

	var copied int
	for {
		l := copy(w.buf[w.pos:], b[copied:])
		copied += l
		if copied == n {
			w.pos += l
			return copied, nil
		}
		w.pos = 0
	}
}

func (w *RingBuffer) Read(b []byte) (int, error) {
	var copied int
	s := w.pos - len(b)
	if s < 0 {
		if n := len(w.buf); n == cap(w.buf) {
			s = max(n+s, w.pos)
			copied = copy(b, w.buf[s:])
		}
		s = 0
	}
	copied += copy(b[copied:], w.buf[s:w.pos])
	return copied, nil
}

func (w *RingBuffer) Clear() {
	w.buf = w.buf[:0]
	w.pos = 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
