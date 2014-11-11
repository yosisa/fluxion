package main

import (
	"bufio"

	"os"
)

type PositionReader struct {
	f   *os.File
	r   *bufio.Reader
	pos int64
	pe  *PositionEntry
}

func NewPositionReader(pe *PositionEntry) (*PositionReader, error) {
	pos := pe.Refresh()
	f, err := os.Open(pe.Path)
	if err != nil {
		return nil, err
	}
	if _, err = f.Seek(pos, os.SEEK_SET); err != nil {
		return nil, err
	}
	r := &PositionReader{
		f:   f,
		r:   bufio.NewReader(f),
		pos: pos,
		pe:  pe,
	}
	return r, nil
}

// ReadLine tries to return a single line, not including the end-of-line bytes.
// It also skip empty line. See also bufio.Reader.ReadLine.
func (r *PositionReader) ReadLine() (line []byte, isPrefix bool, err error) {
	for {
		line, isPrefix, err = r.r.ReadLine()
		if err != nil {
			return
		}
		// If isPrefix is true, it means no the end-of-line bytes skipped.
		// So we must not increment position for that bytes.
		if !isPrefix {
			r.pos++
		}
		if n := len(line); n > 0 {
			r.pos += int64(n)
			r.pe.SetPos(r.pos)
			return
		}
	}
}

func (r *PositionReader) Close() error {
	return r.f.Close()
}
