package main

import (
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/ugorji/go/codec"
	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/message"
	"github.com/yosisa/fluxion/plugin"
)

type CompatibleMode int

const (
	CompatibleDisable CompatibleMode = iota
	CompatibleLegacy
	CompatibleExtend
)

func (m *CompatibleMode) UnmarshalText(b []byte) (err error) {
	switch string(b) {
	case "", "disable":
		*m = CompatibleDisable
	case "legacy":
		*m = CompatibleLegacy
	case "extend":
		*m = CompatibleExtend
	default:
		err = errors.New("CompatibleMode must be disable, legacy or extend")
	}
	return
}

type sequence uint64

func (i *sequence) Next() uint64 {
	return atomic.AddUint64((*uint64)(i), 1)
}

var mh *codec.MsgpackHandle

type Config struct {
	MinWeight  int             `toml:"min_weight"`
	Compatible CompatibleMode  `toml:"compatible"`
	AckTimeout buffer.Duration `toml:"ack_timeout"`
	Servers    []struct {
		Server string `toml:"server"`
		Weight int    `toml:"weight"`
	} `toml:"servers"`
}

type ForwardOutput struct {
	env        *plugin.Env
	conf       *Config
	w          io.Writer
	ackEnabled bool
}

func (o *ForwardOutput) Init(env *plugin.Env) (err error) {
	o.env = env
	o.conf = &Config{}
	if err = env.ReadConfig(o.conf); err != nil {
		return
	}
	if o.conf.MinWeight == 0 {
		o.conf.MinWeight = 5
	}
	if o.conf.Compatible == CompatibleDisable {
		mh = &codec.MsgpackHandle{RawToString: true, WriteExt: true}
	} else {
		mh = &codec.MsgpackHandle{}
	}
	o.ackEnabled = (o.conf.Compatible == CompatibleDisable && o.conf.AckTimeout > 0) ||
		o.conf.Compatible == CompatibleExtend

	var seq sequencer
	if o.ackEnabled {
		var s sequence
		seq = &s
	}

	if len(o.conf.Servers) == 0 {
		return errors.New("No server specified")
	}
	w := NewRoundRobinWriter(o.conf.MinWeight)
	for _, v := range o.conf.Servers {
		f := func(s string) ConnectFunc {
			return func() (net.Conn, error) {
				return net.DialTimeout("tcp", s, 5*time.Second)
			}
		}(v.Server)
		if v.Weight == 0 {
			v.Weight = 60
		}
		w.Add(NewAutoConnectWriter(f, seq, time.Duration(o.conf.AckTimeout)), v.Weight)
	}
	go func() {
		for err := range w.ErrorC {
			o.env.Log.Warning(err)
		}
	}()
	o.w = w
	return
}

func (o *ForwardOutput) Start() (err error) {
	return nil
}

func encode(v interface{}) (b []byte, err error) {
	err = codec.NewEncoderBytes(&b, mh).Encode(v)
	return
}

func decode(b []byte, v interface{}) error {
	return codec.NewDecoderBytes(b, mh).Decode(v)
}

func (o *ForwardOutput) Encode(ev *message.Event) (buffer.Sizer, error) {
	var v []interface{}
	if o.conf.Compatible == CompatibleDisable {
		v = []interface{}{ev.Tag, ev.Time, ev.Record}
	} else {
		v = []interface{}{ev.Tag, ev.Time.Unix(), ev.Record}
	}
	b, err := encode(v)
	if err != nil {
		return nil, err
	}
	if o.ackEnabled {
		// temporary invalid msgpack data, it will be corrected before sending
		b[0] = 0x94
	}
	return buffer.BytesItem(b), nil
}

func (o *ForwardOutput) Write(l []buffer.Sizer) (int, error) {
	for i, b := range l {
		if _, err := o.w.Write(b.(buffer.BytesItem)); err != nil {
			o.env.Log.Error(err)
			return i, err
		}
	}
	return len(l), nil
}

func (o *ForwardOutput) Close() error {
	return nil
}

func main() {
	plugin.New("out-forward", func() plugin.Plugin {
		return &ForwardOutput{}
	}).Run()
}
