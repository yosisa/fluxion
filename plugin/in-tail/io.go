package in_tail

import (
	"bufio"
	"io"
	"os"
)

type PositionReader struct {
	f   *os.File
	r   *bufio.Reader
	pos int64
	pe  *PositionEntry
	buf []byte
	err error
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
// It also skip empty line.
func (r *PositionReader) ReadLine() ([]byte, error) {
	for {
		if r.err != nil {
			return nil, r.readErr()
		}

		line, err := r.r.ReadSlice('\n')
		n := len(line)
		if n == 0 {
			return nil, io.EOF
		}
		r.pos += int64(n)

		if err == io.EOF {
			r.buf = append(r.buf, line...)
			return nil, err
		}
		if err == bufio.ErrBufferFull {
			r.buf = append(r.buf, line...)
			continue
		}
		r.err = err

		r.pe.SetPos(r.pos)
		if n >= 2 && line[n-2] == '\r' {
			line = line[:n-2]
		} else {
			line = line[:n-1]
		}

		if len(r.buf) > 0 {
			line = append(r.buf, line...)
			r.buf = r.buf[:0]
		} else if len(line) == 0 { // empty line
			continue
		}
		return line, nil
	}
}

func (r *PositionReader) Close() error {
	return r.f.Close()
}

func (r *PositionReader) readErr() (err error) {
	if r.err != nil {
		err, r.err = r.err, nil
	}
	return
}
