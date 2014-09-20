package engine

import (
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/pave/process"
)

type Instance struct {
	eng   *Engine
	dec   event.Decoder
	rbuf  *RingBuffer
	units map[int32]*ExecUnit
}

func NewInstance(eng *Engine, cmd *process.Command) *Instance {
	i := &Instance{eng: eng, rbuf: NewRingBuffer(1024), units: make(map[int32]*ExecUnit)}
	cmd.PrepareFunc = func(cmd *exec.Cmd) {
		cmd.Stderr = os.Stderr
		w, _ := cmd.StdinPipe()
		r, _ := cmd.StdoutPipe()
		enc := event.NewEncoder(w)
		i.dec = event.NewDecoder(io.TeeReader(r, i.rbuf))
		for _, u := range i.units {
			u.enc = enc
		}
	}
	return i
}

func (i *Instance) AddExecUnit(id int32, conf map[string]interface{}, bopts *buffer.Options) *ExecUnit {
	unit := &ExecUnit{
		ID:     id,
		Router: &TagRouter{},
		conf:   conf,
		bopts:  bopts,
	}
	i.units[id] = unit
	return unit
}

func (i *Instance) Start() {
	go i.eventLoop()
}

func (i *Instance) eventLoop() {
	for {
		var ev event.Event
		if err := i.dec.Decode(&ev); err != nil {
			b := make([]byte, 1024)
			n, _ := i.rbuf.Read(b)
			log.Fatalf("%v: last read: %x", err, b[:n])
			continue
		}
		i.rbuf.Clear()

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

func (u *ExecUnit) Send(ev *event.Event) error {
	ev.UnitID = u.ID
	return u.enc.Encode(ev)
}
