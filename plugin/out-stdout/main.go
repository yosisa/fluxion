package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/message"
	"github.com/yosisa/fluxion/plugin"
)

type Config struct {
	Format string `toml:"format"`
}

type StdoutOutput struct {
	env  *plugin.Env
	conf *Config
	tmpl *template.Template
}

func (o *StdoutOutput) Name() string {
	return "out-stdout"
}

func (o *StdoutOutput) Init(env *plugin.Env) (err error) {
	o.env = env
	o.conf = &Config{}
	if err = env.ReadConfig(o.conf); err != nil {
		return
	}
	if o.conf.Format != "" {
		o.tmpl, err = template.New("").Parse(o.conf.Format)
	}
	return
}

func (o *StdoutOutput) Start() error {
	return nil
}

func (o *StdoutOutput) Encode(ev *message.Event) (buffer.Sizer, error) {
	var err error
	b := &bytes.Buffer{}
	if o.tmpl != nil {
		err = o.tmpl.Execute(b, ev)
	} else {
		fmt.Fprintf(b, "[%s] %v: ", ev.Tag, ev.Time)
		err = json.NewEncoder(b).Encode(ev.Record)
	}
	if err != nil {
		return nil, err
	}
	return buffer.BytesItem(bytes.TrimRight(b.Bytes(), "\n")), nil
}

func (o *StdoutOutput) Write(l []buffer.Sizer) (int, error) {
	for _, s := range l {
		fmt.Fprintf(os.Stderr, "%s\n", s)
	}
	return len(l), nil
}

func main() {
	plugin.New(func() plugin.Plugin {
		return &StdoutOutput{}
	}).Run()
}
