package main

import (
	"log"
	"os"

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

func (o *StdoutOutput) HandleRecord(r *event.Record) error {
	log.Printf("[%s] %v: %s\n", r.Tag, r.Time, r.Value)
	return nil
}

func main() {
	log.SetOutput(os.Stderr)
	plugin.New(&StdoutOutput{}).Run()
}
