package buffer

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
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
	awake            chan struct{}
	closed           chan struct{}
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
		awake:            make(chan struct{}, 1),
		closed:           make(chan struct{}),
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
	close(m.closed)
	m.m.Lock()
	defer m.m.Unlock()
	for e := m.chunks.Front(); e != nil; e = e.Next() {
		m.handler.Write(e.Value.(*MemoryChunk).Items)
	}
	m.chunks.Init()
}

func (m *Memory) notify() {
	select {
	case m.awake <- struct{}{}:
	default:
	}
}

func (m *Memory) pop() {
	tick := time.Tick(m.flushInterval)
	for {
		select {
		case <-tick:
		case <-m.awake:
		case <-m.closed:
			return
		}

		chunk, n := m.popChunk()
		if chunk == nil {
			continue
		}
		if n > 1 {
			m.notify()
		}

		bt := backOffTick(m.retryInterval, m.maxRetryInterval)
		for {
			select {
			case <-bt.C:
			case <-m.closed:
				return
			}

			n, err := m.handler.Write(chunk.Items)
			if err == nil {
				bt.Stop()
				break
			}
			if n > 0 {
				n = copy(chunk.Items, chunk.Items[n:])
				chunk.Items = chunk.Items[:n]
			}
		}
	}
}

func (m *Memory) popChunk() (*MemoryChunk, int) {
	m.m.Lock()
	defer m.m.Unlock()
	e := m.chunks.Back()
	if e == nil {
		return nil, 0
	}
	c := m.chunks.Remove(e).(*MemoryChunk)
	return c, m.chunks.Len()
}

func backOffTick(initial, max time.Duration) *backoff.Ticker {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = initial
	b.MaxInterval = max
	b.MaxElapsedTime = 0 // infinite
	return backoff.NewTicker(b)
}
