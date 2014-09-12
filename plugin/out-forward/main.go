package main

import (
	"log"
	"os"

	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/fluxion/plugin"
	"github.com/yosisa/gofluent"
)

type Config struct {
	Server string `codec:"server"`
}

type ForwardOutput struct {
	conf   Config
	client *fluent.Client
}

func (o *ForwardOutput) Init(f plugin.ConfigFeeder) error {
	if err := f(&o.conf); err != nil {
		return err
	}
	return nil
}

func (o *ForwardOutput) Start() (err error) {
	o.client, err = fluent.NewClient(fluent.Options{Addr: o.conf.Server})
	return
}

func (o *ForwardOutput) HandleRecord(r *event.Record) error {
	o.client.SendWithTime(r.Tag, r.Time, r.Value)
	return nil
}

func main() {
	log.SetOutput(os.Stderr)
	plugin.New(&ForwardOutput{}).Run()
}
