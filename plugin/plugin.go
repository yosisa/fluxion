package plugin

import (
	"fmt"
	"os"

	"github.com/ugorji/go/codec"
	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/fluxion/pipe"
)

var (
	mh              = &codec.MsgpackHandle{RawToString: true}
	EmbeddedPlugins = make(map[string]PluginFactory)
	writePipe       *pipe.Pipe
)

type PluginFactory func() Plugin

type Env struct {
	ReadConfig func(interface{}) error
	Emit       func(*event.Record)
	Log        *logger
}

type Plugin interface {
	Name() string
	Init(*Env) error
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
	pipe  *pipe.Pipe
}

func New(f PluginFactory) *plugin {
	return &plugin{
		f:     f,
		units: make(map[int32]*execUnit),
	}
}

func (p *plugin) Run() {
	p.pipe = pipe.NewInterProcess(nil, os.Stdout)
	p.eventLoop(pipe.NewInterProcess(os.Stdin, nil))
}

func (p *plugin) RunWithPipe(rp *pipe.Pipe, wp *pipe.Pipe) {
	p.pipe = wp
	p.eventLoop(rp)
}

func (p *plugin) eventLoop(pipe *pipe.Pipe) {
	for ev := range pipe.R {
		unit, ok := p.units[ev.UnitID]
		if !ok {
			unit = newExecUnit(ev.UnitID, p.f(), p.pipe)
			p.units[ev.UnitID] = unit
		}
		unit.eventCh <- ev
	}
}

type execUnit struct {
	ID      int32
	p       Plugin
	eventCh chan *event.Event
	pipe    *pipe.Pipe
	log     *logger
}

func newExecUnit(id int32, p Plugin, pipe *pipe.Pipe) *execUnit {
	u := &execUnit{
		ID:      id,
		p:       p,
		eventCh: make(chan *event.Event, 100),
		pipe:    pipe,
	}
	u.log = &logger{
		Name:     p.Name(),
		Prefix:   fmt.Sprintf("[%02d:%s] ", id, p.Name()),
		emitFunc: u.emit,
	}
	go u.eventLoop()
	return u
}

func (u *execUnit) eventLoop() {
	op, isOutputPlugin := u.p.(OutputPlugin)
	fp, isFilterPlugin := u.p.(FilterPlugin)
	var buf *buffer.Memory
	u.log.Info("plugin started")

	for ev := range u.eventCh {
		switch ev.Name {
		case "set_buffer":
			if isOutputPlugin {
				buf = buffer.NewMemory(ev.Buffer, op)
			}
		case "config":
			b := ev.Payload.([]byte)
			env := &Env{
				ReadConfig: func(v interface{}) error {
					return codec.NewDecoderBytes(b, mh).Decode(v)
				},
				Emit: u.emit,
				Log:  u.log,
			}
			if err := u.p.Init(env); err != nil {
				u.log.Critical("Failed to configure: ", err)
				return
			}
		case "start":
			if err := u.p.Start(); err != nil {
				u.log.Critical("Failed to start: ", err)
				return
			}
		case "record":
			switch {
			case isFilterPlugin:
				r, err := fp.Filter(ev.Record)
				if err != nil {
					u.log.Warning("Filter error: ", err)
					r = ev.Record
				}
				if r != nil {
					u.send(&event.Event{Name: "next_filter", Record: r})
				}
			case isOutputPlugin:
				s, err := op.Encode(ev.Record)
				if err != nil {
					u.log.Warning("Encode error: ", err)
					continue
				}
				if err = buf.Push(s); err != nil {
					u.log.Warning("Buffering error: ", err)
				}
			}
		}
	}
}

func (u *execUnit) emit(record *event.Record) {
	u.send(&event.Event{Name: "record", Record: record})
}

func (u *execUnit) send(ev *event.Event) {
	ev.UnitID = u.ID
	u.pipe.W <- ev
}
