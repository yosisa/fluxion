package main

import (
	"fmt"
	"os"

	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/fluxion/plugin"
)

type StdoutOutput struct {
	env *plugin.Env
}

func (o *StdoutOutput) Name() string {
	return "out-stdout"
}

func (o *StdoutOutput) Init(env *plugin.Env) error {
	o.env = env
	return nil
}

func (o *StdoutOutput) Start() error {
	return nil
}

func (o *StdoutOutput) Encode(r *event.Record) (buffer.Sizer, error) {
	return buffer.StringItem(fmt.Sprintf("[%s] %v: %v\n", r.Tag, r.Time, r.Value)), nil
}

func (o *StdoutOutput) Write(l []buffer.Sizer) (int, error) {
	for _, s := range l {
		fmt.Fprint(os.Stderr, s.(buffer.StringItem))
	}
	return len(l), nil
}

func main() {
	plugin.New(func() plugin.Plugin {
		return &StdoutOutput{}
	}).Run()
}
