package main

import (
	"errors"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"
)

const (
	writeTimeout = 15 * time.Second
	readTimeout  = 15 * time.Second
	readBufSize  = 1024
)

type sequencer interface {
	Next() uint64
}

type ConnectFunc func() (net.Conn, error)

type AutoConnectWriter struct {
	c          net.Conn
	connect    ConnectFunc
	seq        sequencer
	ackTimeout time.Duration
	ackWaiters map[interface{}]chan struct{}
	err        error
	m          sync.RWMutex
	closed     chan struct{}
	closeOnce  sync.Once
}

func NewAutoConnectWriter(f ConnectFunc, seq sequencer, ackTimeout time.Duration) *AutoConnectWriter {
	return &AutoConnectWriter{
		connect:    f,
		seq:        seq,
		ackTimeout: ackTimeout,
		ackWaiters: make(map[interface{}]chan struct{}),
	}
}

func (w *AutoConnectWriter) init() error {
	c, err := w.connect()
	if err != nil {
		return err
	}
	w.c = c
	w.closed = make(chan struct{})
	w.closeOnce = sync.Once{}
	go w.reader()
	return nil
}

func (w *AutoConnectWriter) Write(b []byte) (n int, err error) {
	if err = w.readError(); err != nil {
		w.setError(nil)
		w.Close()
		return
	}
	if w.c == nil {
		if err = w.init(); err != nil {
			return
		}
	}

	if w.seq != nil {
		var option []byte
		id := w.seq.Next()
		if option, err = encode(map[string]interface{}{"chunk": id}); err != nil {
			return
		}
		b = append(b, option...)
		if w.ackTimeout > 0 {
			waiter := w.setAckWaiter(id)
			defer func() {
				if err != nil {
					return
				}
				select {
				case <-waiter:
					return
				case <-w.closed:
					err = errors.New("Unexpected close")
				case <-time.After(w.ackTimeout):
					err = errors.New("ack not returned")
					w.Close()
				}
				w.unsetAckWaiter(id)
				n = 0
			}()
		}
	}

	w.c.SetWriteDeadline(time.Now().Add(writeTimeout))
	if n, err = w.c.Write(b); err != nil {
		w.Close()
	}
	return
}

func (w *AutoConnectWriter) reader() {
	b := make([]byte, readBufSize)
	for {
		n, err := w.c.Read(b)
		if err != nil {
			w.setError(err)
			return
		}
		var ack struct {
			Ack uint64 `json:"ack"`
		}
		if err = decode(b[:n], &ack); err != nil {
			w.setError(err)
			return
		}
		w.notifyAck(ack.Ack)
		b = b[:readBufSize]
	}
}

func (w *AutoConnectWriter) setAckWaiter(id interface{}) chan struct{} {
	w.m.Lock()
	defer w.m.Unlock()
	c := make(chan struct{})
	w.ackWaiters[id] = c
	return c
}

func (w *AutoConnectWriter) unsetAckWaiter(id interface{}) {
	w.m.Lock()
	defer w.m.Unlock()
	delete(w.ackWaiters, id)
}

func (w *AutoConnectWriter) notifyAck(id interface{}) {
	w.m.Lock()
	defer w.m.Unlock()
	if c, ok := w.ackWaiters[id]; ok {
		close(c)
		delete(w.ackWaiters, id)
	}
}

func (w *AutoConnectWriter) setError(err error) {
	w.m.Lock()
	defer w.m.Unlock()
	w.err = err
}

func (w *AutoConnectWriter) readError() error {
	w.m.RLock()
	defer w.m.RUnlock()
	return w.err
}

func (w *AutoConnectWriter) Close() {
	w.closeOnce.Do(func() {
		w.c.Close()
		close(w.closed)
		w.c = nil
	})
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
