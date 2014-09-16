package plugin

import (
	"io"
	"log"
	"os"

	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/engine"
	"github.com/yosisa/fluxion/event"
)

type ConfigFeeder func(interface{}) error

type Plugin interface {
	Init(ConfigFeeder) error
	Start() error
}

type OutputPlugin interface {
	Plugin
	Encode(*event.Record) (buffer.Sizer, error)
	Write([]buffer.Sizer) (int, error)
}

type plugin struct {
	p Plugin
}

func New(p Plugin) *plugin {
	return &plugin{p: p}
}

func (p *plugin) Run() {
	p.eventListener()
}

func (p *plugin) eventListener() {
	dec := event.NewDecoder(os.Stdin)
	op, isOutputPlugin := p.p.(OutputPlugin)
	var buf *buffer.Memory

	for {
		var ev event.Event
		if err := dec.Decode(&ev); err != nil {
			if err == io.EOF {
				return
			} else {
				log.Fatal(err)
			}
		}

		switch ev.Name {
		case "set_buffer":
			if isOutputPlugin {
				buf = buffer.NewMemory(ev.Buffer, op)
			}
		case "config":
			b := ev.Payload.([]byte)
			f := func(v interface{}) error {
				return engine.Decode(b, v)
			}
			if err := p.p.Init(f); err != nil {
				log.Fatal(err)
			}
		case "start":
			if err := p.p.Start(); err != nil {
				log.Fatal(err)
			}
		case "record":
			if isOutputPlugin {
				s, err := op.Encode(ev.Record)
				if err != nil {
					log.Fatal(err)
				}
				if err = buf.Push(s); err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}

var encoder = event.NewEncoder(os.Stdout)

func Emit(record *event.Record) error {
	ev := &event.Event{Name: "record", Record: record}
	return encoder.Encode(ev)
}
