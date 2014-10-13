package plugin

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ugorji/go/codec"
	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/log"
	"github.com/yosisa/fluxion/message"
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
	Emit       func(*message.Event)
	Log        *log.Logger
}

type Plugin interface {
	Name() string
	Init(*Env) error
	Start() error
}

type OutputPlugin interface {
	Plugin
	Encode(*message.Event) (buffer.Sizer, error)
	Write([]buffer.Sizer) (int, error)
}

type FilterPlugin interface {
	Plugin
	Filter(*message.Event) (*message.Event, error)
}

type plugin struct {
	f     PluginFactory
	units map[int32]*execUnit
	pipe  pipe.Pipe
}

func New(f PluginFactory) *plugin {
	return &plugin{
		f:     f,
		units: make(map[int32]*execUnit),
	}
}

func (p *plugin) Run() {
	p.pipe = pipe.NewInterProcess(nil, os.Stdout)
	go p.signalHandler()
	p.eventLoop(pipe.NewInterProcess(os.Stdin, nil))
}

func (p *plugin) RunWithPipe(rp pipe.Pipe, wp pipe.Pipe) {
	p.pipe = wp
	p.eventLoop(rp)
}

func (p *plugin) eventLoop(pipe pipe.Pipe) {
	for {
		m, err := pipe.Read()
		if err != nil {
			return
		}

		switch m.Type {
		case message.TypInfoRequest:
			p.pipe.Write(&message.Message{
				Type:    message.TypInfoResponse,
				Payload: &message.PluginInfo{ProtoVer: 1},
			})
		case message.TypStop:
			p.stop()
			p.pipe.Write(&message.Message{Type: message.TypTerminated})
			return
		default:
			unit, ok := p.units[m.UnitID]
			if !ok {
				unit = newExecUnit(m.UnitID, p.f(), p.pipe)
				p.units[m.UnitID] = unit
			}
			unit.msgC <- m
		}
	}
}

func (p *plugin) stop() {
	var wg sync.WaitGroup
	for _, unit := range p.units {
		wg.Add(1)
		go func(unit *execUnit) {
			unit.stop()
			wg.Done()
		}(unit)
	}
	wg.Wait()
}

func (p *plugin) signalHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT)
	for _ = range c {
	}
}

type execUnit struct {
	ID    int32
	p     Plugin
	msgC  chan *message.Message
	doneC chan bool
	pipe  pipe.Pipe
	log   *log.Logger
}

func newExecUnit(id int32, p Plugin, pipe pipe.Pipe) *execUnit {
	u := &execUnit{
		ID:    id,
		p:     p,
		msgC:  make(chan *message.Message, 100),
		doneC: make(chan bool),
		pipe:  pipe,
	}
	u.log = &log.Logger{
		Name:     p.Name(),
		Prefix:   fmt.Sprintf("[%02d:%s] ", id, p.Name()),
		EmitFunc: u.emit,
	}
	go u.eventLoop()
	return u
}

func (u *execUnit) eventLoop() {
	op, isOutputPlugin := u.p.(OutputPlugin)
	fp, isFilterPlugin := u.p.(FilterPlugin)
	var buf *buffer.Memory
	u.log.Info("plugin started")

	for m := range u.msgC {
		switch m.Type {
		case message.TypBufferOption:
			if isOutputPlugin {
				buf = buffer.NewMemory(m.Payload.(*buffer.Options), op)
			}
		case message.TypConfigure:
			b := m.Payload.([]byte)
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
		case message.TypStart:
			if err := u.p.Start(); err != nil {
				u.log.Critical("Failed to start: ", err)
				return
			}
		case message.TypEvent:
			switch {
			case isFilterPlugin:
				ev := m.Payload.(*message.Event)
				r, err := fp.Filter(ev)
				if err != nil {
					u.log.Warning("Filter error: ", err)
					r = ev
				}
				if r != nil {
					u.send(&message.Message{Type: message.TypEventChain, Payload: r})
				}
			case isOutputPlugin:
				s, err := op.Encode(m.Payload.(*message.Event))
				if err != nil {
					u.log.Warning("Encode error: ", err)
					continue
				}
				if err = buf.Push(s); err != nil {
					u.log.Warning("Buffering error: ", err)
				}
			}
		case message.TypStop:
			if isOutputPlugin {
				buf.Close()
			}
		}
	}
	close(u.doneC)
}

func (u *execUnit) emit(ev *message.Event) {
	u.send(&message.Message{Type: message.TypEvent, Payload: ev})
}

func (u *execUnit) send(m *message.Message) {
	m.UnitID = u.ID
	u.pipe.Write(m)
}

func (u *execUnit) stop() {
	u.msgC <- &message.Message{Type: message.TypStop}
	close(u.msgC)
	<-u.doneC
}
