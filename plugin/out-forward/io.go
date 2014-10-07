package main

import "io"

type ConnectFunc func() (io.Writer, error)

type AutoConnectWriter struct {
	w       io.Writer
	connect func() (io.Writer, error)
}

func NewAutoConnectWriter(f ConnectFunc) *AutoConnectWriter {
	return &AutoConnectWriter{connect: f}
}

func (w *AutoConnectWriter) Write(b []byte) (int, error) {
	if w.w == nil {
		writer, err := w.connect()
		if err != nil {
			return 0, err
		}
		w.w = writer
	}

	n, err := w.w.Write(b)
	if err != nil {
		if closer, ok := w.w.(io.Closer); ok {
			closer.Close()
		}
		w.w = nil
	}
	return n, err
}
