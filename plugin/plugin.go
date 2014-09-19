package plugin

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

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

type FilterPlugin interface {
	Plugin
	Filter(*event.Record) (*event.Record, error)
}

type plugin struct {
	p Plugin
}

func New(p Plugin) *plugin {
	Log.Name = strings.SplitN(os.Args[0], "-", 2)[1]
	Log.Prefix = fmt.Sprintf("[%s] ", Log.Name)
	return &plugin{p: p}
}

func (p *plugin) Run() {
	Log.Infof("Plugin started")
	p.eventListener()
}

func (p *plugin) eventListener() {
	dec := event.NewDecoder(os.Stdin)
	op, isOutputPlugin := p.p.(OutputPlugin)
	fp, isFilterPlugin := p.p.(FilterPlugin)
	var buf *buffer.Memory

	for {
		var ev event.Event
		if err := dec.Decode(&ev); err != nil {
			if err == io.EOF {
				return
			} else {
				Log.Warning(err)
				continue
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
				Log.Fatal("Failed to configure: ", err)
			}
		case "start":
			if err := p.p.Start(); err != nil {
				Log.Fatal("Failed to start: ", err)
			}
		case "record":
			switch {
			case isFilterPlugin:
				r, err := fp.Filter(ev.Record)
				if err != nil {
					Log.Warning("Filter error: ", err)
					continue
				}
				ev := &event.Event{Name: "next_filter", Record: r}
				mutex.Lock()
				if err = encoder.Encode(ev); err != nil {
					Log.Warning("Failed to transmit record: ", err)
				}
				mutex.Unlock()
			case isOutputPlugin:
				s, err := op.Encode(ev.Record)
				if err != nil {
					Log.Warning("Encode error: ", err)
					continue
				}
				if err = buf.Push(s); err != nil {
					Log.Warning("Buffering error: ", err)
				}
			}
		}
	}
}

var encoder = event.NewEncoder(os.Stdout)
var mutex sync.Mutex

func Emit(record *event.Record) error {
	ev := &event.Event{Name: "record", Record: record}
	mutex.Lock()
	defer mutex.Unlock()
	return encoder.Encode(ev)
}
