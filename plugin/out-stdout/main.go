package main

import (
	"fmt"
	"log"
	"os"

	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/fluxion/plugin"
)

type StdoutOutput struct {
}

func (o *StdoutOutput) Init(f plugin.ConfigFeeder) error {
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
		log.Print(s.(buffer.StringItem))
	}
	return len(l), nil
}

func main() {
	log.SetOutput(os.Stderr)
	plugin.New(&StdoutOutput{}).Run()
}
