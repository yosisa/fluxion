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
	Name             string    `toml:"name" codec:"name"`
	Type             string    `toml:"type" codec:"type"`
	MaxChunkSize     HumanSize `toml:"max_chunk_size" codec:"max_chunk_size"`
	MaxQueueSize     HumanSize `toml:"max_queue_size" codec:"max_queue_size"`
	FlushInterval    Duration  `toml:"flush_interval" codec:"flush_interval"`
	RetryInterval    Duration  `toml:"retry_interval" codec:"retry_interval"`
	MaxRetryInterval Duration  `toml:"max_retry_interval" codec:"max_retry_interval"`
}

func (o *Options) SetDefault() {
	if o.Name == "" {
		o.Name = "default"
	}
	if o.Type == "" {
		o.Type = "memory"
	}
	if o.MaxChunkSize == 0 {
		o.MaxChunkSize = 1024 * 1024
	}
	if o.MaxQueueSize == 0 {
		o.MaxQueueSize = 256
	}
	if o.FlushInterval == 0 {
		o.FlushInterval = Duration(0)
	}
	if o.RetryInterval == 0 {
		o.RetryInterval = Duration(100 * time.Millisecond)
	}
	if o.MaxRetryInterval == 0 {
		o.MaxRetryInterval = Duration(time.Minute)
	}
}
