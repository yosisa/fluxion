package plugin

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/engine"
	"github.com/yosisa/fluxion/event"
)

var eventCh = make(chan *event.Event, 100)

type ConfigFeeder func(interface{}) error

type PluginFactory func() Plugin

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
	f     PluginFactory
	units map[int32]*execUnit
}

func New(f PluginFactory) *plugin {
	Log.Name = strings.SplitN(os.Args[0], "-", 2)[1]
	Log.Prefix = fmt.Sprintf("[%s] ", Log.Name)
	return &plugin{
		f:     f,
		units: make(map[int32]*execUnit),
	}
}

func (p *plugin) Run() {
	Log.Infof("Plugin created")
	go eventSender()
	p.eventListener()
}

func (p *plugin) eventListener() {
	dec := event.NewDecoder(os.Stdin)
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

		unit, ok := p.units[ev.UnitID]
		if !ok {
			unit = newExecUnit(ev.UnitID, p.f())
			p.units[ev.UnitID] = unit
		}
		unit.eventCh <- &ev
	}
}

type execUnit struct {
	ID      int32
	p       Plugin
	eventCh chan *event.Event
}

func newExecUnit(id int32, p Plugin) *execUnit {
	u := &execUnit{
		ID:      id,
		p:       p,
		eventCh: make(chan *event.Event, 100),
	}
	go u.eventLoop()
	return u
}

func (u *execUnit) eventLoop() {
	op, isOutputPlugin := u.p.(OutputPlugin)
	fp, isFilterPlugin := u.p.(FilterPlugin)
	var buf *buffer.Memory

	for ev := range u.eventCh {
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
			if err := u.p.Init(f); err != nil {
				Log.Critical("Failed to configure: ", err)
				return
			}
		case "start":
			if err := u.p.Start(); err != nil {
				Log.Critical("Failed to start: ", err)
				return
			}
		case "record":
			switch {
			case isFilterPlugin:
				r, err := fp.Filter(ev.Record)
				if err != nil {
					Log.Warning("Filter error: ", err)
					r = ev.Record
				}
				if r != nil {
					send(&event.Event{UnitID: u.ID, Name: "next_filter", Record: r})
				}
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

func Emit(record *event.Record) {
	send(&event.Event{Name: "record", Record: record})
}

func send(ev *event.Event) {
	eventCh <- ev
}

func eventSender() {
	encoder := event.NewEncoder(os.Stdout)
	for ev := range eventCh {
		if err := encoder.Encode(ev); err != nil {
			if err == io.EOF {
				return
			}
			Log.Warning("Failed to send event: ", err)
		}
	}
}
