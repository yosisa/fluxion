package engine

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/fluxion/pipe"
)

type Instance struct {
	eng   *Engine
	dec   event.Decoder
	units map[int32]*ExecUnit
	rp    *pipe.Pipe
	wp    *pipe.Pipe
}

func NewInstance(eng *Engine) *Instance {
	return &Instance{
		eng:   eng,
		units: make(map[int32]*ExecUnit),
	}
}

func (i *Instance) AddExecUnit(id int32, conf map[string]interface{}, bopts *buffer.Options) *ExecUnit {
	unit := &ExecUnit{
		ID:     id,
		Router: &TagRouter{},
		conf:   conf,
		bopts:  bopts,
		pipe:   i.wp,
	}
	i.units[id] = unit
	return unit
}

func (i *Instance) Start() {
	go i.eventLoop()
}

func (i *Instance) eventLoop() {
	for ev := range i.rp.R {
		switch ev.Name {
		case "record":
			i.eng.Filter(ev.Record)
		case "next_filter":
			unit, ok := i.units[ev.UnitID]
			if !ok {
				log.Printf("Unit ID %d not known", ev.UnitID)
				continue
			}

			if e := unit.Router.Route(ev.Record.Tag); e != nil {
				e.Emit(ev.Record)
			} else {
				i.eng.Emit(ev.Record)
			}
		}
	}
}

type ExecUnit struct {
	ID     int32
	Router *TagRouter
	enc    event.Encoder
	conf   map[string]interface{}
	bopts  *buffer.Options
	pipe   *pipe.Pipe
}

func (u *ExecUnit) SetBuffer() error {
	return u.Send(&event.Event{Name: "set_buffer", Buffer: u.bopts})
}

func (u *ExecUnit) Configure() error {
	b, err := Encode(u.conf)
	if err != nil {
		return err
	}
	return u.Send(&event.Event{Name: "config", Payload: b})
}

func (u *ExecUnit) Start() error {
	return u.Send(&event.Event{Name: "start"})
}

func (u *ExecUnit) Emit(record *event.Record) error {
	return u.Send(&event.Event{Name: "record", Record: record})
}

func (u *ExecUnit) Send(ev *event.Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	ev.UnitID = u.ID
	u.pipe.W <- ev
	return
}

func prepareFuncFactory(i *Instance) func(*exec.Cmd) {
	return func(cmd *exec.Cmd) {
		cmd.Stderr = os.Stderr
		w, _ := cmd.StdinPipe()
		r, _ := cmd.StdoutPipe()
		i.rp = pipe.NewInterProcess(r, nil)
		i.wp = pipe.NewInterProcess(nil, w)
		for _, u := range i.units {
			u.pipe = i.wp
		}
	}
}
