package buffer

import (
	"strconv"
	"time"
)

type HumanSize int64

func (s *HumanSize) UnmarshalText(b []byte) error {
	var multiplier int64 = 1000
	if b[len(b)-1] == 'i' {
		multiplier = 1024
		b = b[:len(b)-1]
	}

	var n int64 = 1
	switch b[len(b)-1] {
	case 'K', 'k':
		n = multiplier
		b = b[:len(b)-1]
	case 'M', 'm':
		n = multiplier * multiplier
		b = b[:len(b)-1]
	case 'G', 'g':
		n = multiplier * multiplier * multiplier
		b = b[:len(b)-1]
	case 'T', 't':
		n = multiplier * multiplier * multiplier * multiplier
		b = b[:len(b)-1]
	}

	i, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return err
	}
	*s = HumanSize(i * n)
	return nil
}

type Duration time.Duration

func (d *Duration) UnmarshalText(b []byte) error {
	if len(b) == 0 {
		*d = 0
		return nil
	}

	du, err := time.ParseDuration(string(b))
	if err != nil {
		return err
	}
	*d = Duration(du)
	return nil
}

type Options struct {
	Name          string    `toml:"name" codec:"name"`
	Type          string    `toml:"type" codec:"type"`
	MaxChunkSize  HumanSize `toml:"max_chunk_size" codec:"max_chunk_size"`
	MaxQueueSize  HumanSize `toml:"max_queue_size" codec:"max_queue_size"`
	FlushInterval Duration  `toml:"flush_interval" codec:"flush_interval"`
}

var DefaultOptions = &Options{
	Name:          "default",
	Type:          "memory",
	MaxChunkSize:  8 * 1024 * 1024,
	MaxQueueSize:  64,
	FlushInterval: Duration(15 * time.Second),
}
