package main

import (
	"errors"
	"io"
	"net"
	"time"

	"github.com/ugorji/go/codec"
	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/message"
	"github.com/yosisa/fluxion/plugin"
)

type Config struct {
	MinWeight  int  `toml:"min_weight"`
	Compatible bool `toml:"compatible"`

	Servers []struct {
		Server string `toml:"server"`
		Weight int    `toml:"weight"`
	} `toml:"servers"`
}

type ForwardOutput struct {
	env  *plugin.Env
	conf *Config
	conn net.Conn
	w    io.Writer
	mh   *codec.MsgpackHandle
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
	if o.conf.Compatible {
		o.mh = &codec.MsgpackHandle{}
	} else {
		o.mh = &codec.MsgpackHandle{RawToString: true, WriteExt: true}
	}

	if len(o.conf.Servers) == 0 {
		return errors.New("No server specified")
	}
	w := NewRoundRobinWriter(o.conf.MinWeight)
	for _, v := range o.conf.Servers {
		f := func(s string) ConnectFunc {
			return func() (io.Writer, error) {
				return net.DialTimeout("tcp", s, 5*time.Second)
			}
		}(v.Server)
		if v.Weight == 0 {
			v.Weight = 60
		}
		w.Add(NewAutoConnectWriter(f), v.Weight)
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

func (o *ForwardOutput) Encode(ev *message.Event) (buffer.Sizer, error) {
	var b []byte
	var v []interface{}
	if o.conf.Compatible {
		v = []interface{}{ev.Tag, ev.Time.Unix(), ev.Record}
	} else {
		v = []interface{}{ev.Tag, ev.Time, ev.Record}
	}
	if err := codec.NewEncoderBytes(&b, o.mh).Encode(v); err != nil {
		return nil, err
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
