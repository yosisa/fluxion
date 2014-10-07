package main

import (
	"io"
	"math/rand"
)

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

type RoundRobinWriter struct {
	writers []io.Writer
	weights []int
	total   int
}

func (w *RoundRobinWriter) Add(writer io.Writer, weight int) {
	w.writers = append(w.writers, writer)
	w.weights = append(w.weights, weight)
	w.total += weight
}

func (w *RoundRobinWriter) Write(b []byte) (n int, err error) {
	i := w.choice()
	for attempts := 0; attempts < len(w.writers); attempts++ {
		n, err = w.writers[i].Write(b)
		if err == nil {
			return
		}
		i++
		if i >= len(w.writers) {
			i = 0
		}
	}
	return 0, err
}

func (w *RoundRobinWriter) choice() int {
	n := rand.Intn(w.total)
	for i := 0; i < len(w.weights); i++ {
		n -= w.weights[i]
		if n < 0 {
			return i
		}
	}
	return 0
}
