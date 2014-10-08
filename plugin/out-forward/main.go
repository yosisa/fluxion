package main

import (
	"errors"
	"io"
	"net"
	"time"

	"github.com/ugorji/go/codec"
	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/fluxion/plugin"
)

var mh = &codec.MsgpackHandle{}

type Config struct {
	MinWeight int `codec:"min_weight"`

	Servers []struct {
		Server string `codec:"server"`
		Weight int    `codec:"weight"`
	} `codec:"servers"`
}

type ForwardOutput struct {
	env  *plugin.Env
	conf *Config
	conn net.Conn
	w    io.Writer
}

func (o *ForwardOutput) Name() string {
	return "out-forward"
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

func (o *ForwardOutput) Encode(r *event.Record) (buffer.Sizer, error) {
	var b []byte
	v := []interface{}{r.Tag, r.Time.Unix(), r.Value}
	if err := codec.NewEncoderBytes(&b, mh).Encode(v); err != nil {
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

func main() {
	plugin.New(func() plugin.Plugin {
		return &ForwardOutput{}
	}).Run()
}
