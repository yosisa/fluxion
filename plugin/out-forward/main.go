package main

import (
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
	Server string `codec:"server"`
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
	o.w = NewAutoConnectWriter(func() (io.Writer, error) {
		return net.DialTimeout("tcp", o.conf.Server, 5*time.Second)
	})
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
