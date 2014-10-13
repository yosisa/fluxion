package engine

import (
	"container/list"
	"log"
	"os"
	"os/exec"

	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/message"
	"github.com/yosisa/fluxion/pipe"
)

type Instance struct {
	name  string
	eng   *Engine
	dec   message.Decoder
	units map[int32]*ExecUnit
	rp    pipe.Pipe
	wp    pipe.Pipe
	doneC chan bool
}

func NewInstance(name string, eng *Engine) *Instance {
	return &Instance{
		name:  name,
		eng:   eng,
		units: make(map[int32]*ExecUnit),
		doneC: make(chan bool),
	}
}

func (i *Instance) AddExecUnit(id int32, conf map[string]interface{}, bopts *buffer.Options) *ExecUnit {
	i.units[id] = newExecUnit(id, conf, bopts)
	return i.units[id]
}

func (i *Instance) Start() {
	i.wp.Write(&message.Message{Type: message.TypInfoRequest})
	go i.eventLoop()
}

func (i *Instance) Stop() {
	i.wp.Write(&message.Message{Type: message.TypStop})
	<-i.doneC
}

func (i *Instance) eventLoop() {
	for {
		m, err := i.rp.Read()
		if err != nil {
			return
		}

		switch m.Type {
		case message.TypInfoResponse:
			info := m.Payload.(*message.PluginInfo)
			i.eng.log.Infof("%s plugin: protocol version %d", i.name, info.ProtoVer)
			for _, u := range i.units {
				u.pipe = i.wp
				u.Start()
			}
		case message.TypEvent:
			i.eng.Filter(m.Payload.(*message.Event))
		case message.TypEventChain:
			unit, ok := i.units[m.UnitID]
			if !ok {
				log.Printf("Unit ID %d not known", m.UnitID)
				continue
			}

			ev := m.Payload.(*message.Event)
			if e := unit.Router.Route(ev.Tag); e != nil {
				e.Emit(ev)
			} else {
				i.eng.Emit(ev)
			}
		case message.TypTerminated:
			close(i.doneC)
			return
		}
	}
}

type ExecUnit struct {
	ID      int32
	Router  *TagRouter
	enc     message.Encoder
	conf    map[string]interface{}
	bopts   *buffer.Options
	pipe    pipe.Pipe
	pending *pending
	term    int
	emitC   chan *message.Message
}

func newExecUnit(id int32, conf map[string]interface{}, bopts *buffer.Options) *ExecUnit {
	u := &ExecUnit{
		ID:      id,
		Router:  &TagRouter{},
		conf:    conf,
		bopts:   bopts,
		pending: newPending(100 * 1024),
		emitC:   make(chan *message.Message),
	}
	go u.pendingLoop()
	return u
}

func (u *ExecUnit) Start() error {
	if err := u.Send(&message.Message{Type: message.TypBufferOption, Payload: u.bopts}); err != nil {
		return err
	}

	b, err := Encode(u.conf)
	if err != nil {
		return err
	}
	if err := u.Send(&message.Message{Type: message.TypConfigure, Payload: b}); err != nil {
		return err
	}

	if err := u.Send(&message.Message{Type: message.TypStart}); err != nil {
		return err
	}

	u.term++
	return nil
}

func (u *ExecUnit) Emit(ev *message.Event) error {
	u.emitC <- &message.Message{Type: message.TypEvent, Payload: ev}
	return nil
}

func (u *ExecUnit) pendingLoop() {
	term := u.term
	for {
		curTerm := u.term
		if curTerm > term {
			term = curTerm
			err := u.pending.Flush(u.sendPending)
			if err == nil {
				u.emitLoop()
				continue
			}
		}

		ev := <-u.emitC
		u.pending.Add(ev)
	}
}

func (u *ExecUnit) emitLoop() {
	for {
		ev := <-u.emitC
		err := u.Send(ev)
		if err == nil {
			continue
		}

		u.pending.Add(ev)
		return
	}
}

func (u *ExecUnit) sendPending(v interface{}) error {
	return u.Send(v.(*message.Message))
}

func (u *ExecUnit) Send(ev *message.Message) (err error) {
	ev.UnitID = u.ID
	return u.pipe.Write(ev)
}

type pending struct {
	list  *list.List
	limit int
}

func newPending(limit int) *pending {
	return &pending{
		list:  list.New(),
		limit: limit,
	}
}

func (p *pending) Add(v interface{}) {
	// trim the pending if limit exceeded
	for i := p.list.Len() - p.limit; i >= 0; i-- {
		p.list.Remove(p.list.Front())
	}
	p.list.PushBack(v)
}

func (p *pending) Flush(f func(interface{}) error) error {
	for e := p.list.Front(); e != nil; e = p.list.Front() {
		if err := f(e.Value); err != nil {
			return err
		}
		p.list.Remove(e)
	}
	return nil
}

func prepareFuncFactory(i *Instance) func(*exec.Cmd) {
	return func(cmd *exec.Cmd) {
		cmd.Stderr = os.Stderr
		w, _ := cmd.StdinPipe()
		r, _ := cmd.StdoutPipe()
		i.rp = pipe.NewInterProcess(r, nil)
		i.wp = pipe.NewInterProcess(nil, w)
		i.Start()
	}
}
