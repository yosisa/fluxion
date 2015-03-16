package buffer

import (
	"container/list"
	"fmt"
	"math"
	"sync"
	"time"
)

type StringItem string

func (s StringItem) Size() int64 {
	return int64(len(s))
}

type BytesItem []byte

func (s BytesItem) Size() int64 {
	return int64(len(s))
}

type Sizer interface {
	Size() int64
}

type Chunk interface {
	Push(Sizer)
}

type Handler interface {
	Write([]Sizer) (int, error)
}

type ChunkHandler func(c Chunk) error

type MemoryChunk struct {
	Size  int64
	Items []Sizer
}

func (m *MemoryChunk) Push(s Sizer) {
	m.Size += s.Size()
	m.Items = append(m.Items, s)
}

type Memory struct {
	chunks           *list.List
	maxChunkSize     int64
	maxQueueSize     int64
	flushInterval    time.Duration
	retryInterval    time.Duration
	maxRetryInterval time.Duration
	handler          Handler
	eventCh          chan bool
	closed           bool
	m                sync.Mutex
}

func NewMemory(opts *Options, h Handler) *Memory {
	m := &Memory{
		chunks:           list.New(),
		maxChunkSize:     int64(opts.MaxChunkSize),
		maxQueueSize:     int64(opts.MaxQueueSize),
		flushInterval:    time.Duration(opts.FlushInterval),
		retryInterval:    time.Duration(opts.RetryInterval),
		maxRetryInterval: time.Duration(opts.MaxRetryInterval),
		handler:          h,
		eventCh:          make(chan bool),
	}
	go m.pop()
	return m
}

func (m *Memory) Push(s Sizer) error {
	n := s.Size()
	if n > m.maxChunkSize {
		return fmt.Errorf("Too large item: %d, max: %d", n, m.maxChunkSize)
	}
	m.m.Lock()
	defer m.m.Unlock()

	e := m.chunks.Front()
	if e == nil || e.Value.(*MemoryChunk).Size+n > m.maxChunkSize {
		if e != nil {
			m.notify()
		}

		e = m.chunks.PushFront(&MemoryChunk{})
		if int64(m.chunks.Len()) > m.maxQueueSize {
			m.chunks.Remove(m.chunks.Back())
		}
	}

	e.Value.(*MemoryChunk).Push(s)
	if m.flushInterval == 0 {
		m.notify()
	}
	return nil
}

func (m *Memory) Close() {
	m.closed = true
	close(m.eventCh)
	m.m.Lock()
	defer m.m.Unlock()
	for e := m.chunks.Front(); e != nil; e = e.Next() {
		m.handler.Write(e.Value.(*MemoryChunk).Items)
	}
	m.chunks.Init()
}

func (m *Memory) notify() {
	select {
	case m.eventCh <- true:
	default:
	}
}

func (m *Memory) pop() {
	var attempts int
	var chunk *MemoryChunk
	tick := time.Tick(m.flushInterval)
	for {
		select {
		case <-m.eventCh:
			if m.closed {
				return
			} else if attempts > 0 {
				continue
			}
		case <-tick:
		}

		if chunk == nil {
			if chunk = m.popChunk(); chunk == nil {
				continue
			}
		}

		n, err := m.handler.Write(chunk.Items)
		if err != nil {
			n = copy(chunk.Items, chunk.Items[n:])
			chunk.Items = chunk.Items[:n]
			tick = time.Tick(backoff(m.retryInterval, attempts, m.maxRetryInterval))
			attempts++
		} else {
			chunk = nil
			if attempts > 0 {
				attempts = 0
				tick = time.Tick(m.flushInterval)
			}
		}
	}
}

func (m *Memory) popChunk() *MemoryChunk {
	m.m.Lock()
	defer m.m.Unlock()
	e := m.chunks.Back()
	if e == nil {
		return nil
	}
	return m.chunks.Remove(e).(*MemoryChunk)
}

func backoff(d time.Duration, count int, max time.Duration) time.Duration {
	r := d * time.Duration(math.Exp2(float64(count)))
	if r > max {
		return max
	}
	return r
}
