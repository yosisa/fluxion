package main

import (
	"errors"
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

type weightedWriter struct {
	io.Writer
	weight int
	max    int
}

type RoundRobinWriter struct {
	ErrorC    chan error
	writers   []*weightedWriter
	total     int
	minWeight int
}

func NewRoundRobinWriter(minWeight int) *RoundRobinWriter {
	return &RoundRobinWriter{
		ErrorC:    make(chan error, 10),
		minWeight: minWeight,
	}
}

func (w *RoundRobinWriter) Add(writer io.Writer, weight int) {
	w.writers = append(w.writers, &weightedWriter{writer, weight, weight})
	w.total += weight
}

func (w *RoundRobinWriter) Write(b []byte) (n int, err error) {
	i := w.choice()
	for attempts := 0; attempts < len(w.writers); attempts++ {
		n, err = w.writers[i].Write(b)
		if err == nil {
			// Restore weight
			if w.writers[i].weight != w.writers[i].max {
				w.total += w.writers[i].max - w.writers[i].weight
				w.writers[i].weight = w.writers[i].max
			}
			return
		}
		w.error(err)

		// Decrease weight so that avoid retrying too often
		weight := w.writers[i].weight / 2
		if weight < w.minWeight {
			weight = w.minWeight
		}
		w.total -= w.writers[i].weight - weight
		w.writers[i].weight = weight

		i++
		if i >= len(w.writers) {
			i = 0
		}
	}
	return 0, errors.New("No server is available")
}

func (w *RoundRobinWriter) choice() int {
	n := rand.Intn(w.total)
	for i := 0; i < len(w.writers); i++ {
		n -= w.writers[i].weight
		if n < 0 {
			return i
		}
	}
	w.error(errors.New("Failed to choice a random server"))
	return 0
}

func (w *RoundRobinWriter) error(err error) {
	select {
	case w.ErrorC <- err:
	default:
	}
}
