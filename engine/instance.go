package engine

import (
	"os"
	"os/exec"

	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/pave/process"
)

type Instance struct {
	eng  *Engine
	enc  event.Encoder
	dec  event.Decoder
	conf map[string]interface{}
}

func NewInstance(eng *Engine, cmd *process.Command, conf map[string]interface{}) *Instance {
	i := &Instance{eng: eng, conf: conf}
	cmd.PrepareFunc = func(cmd *exec.Cmd) {
		cmd.Stderr = os.Stderr
		w, _ := cmd.StdinPipe()
		r, _ := cmd.StdoutPipe()
		i.enc = event.NewEncoder(w)
		i.dec = event.NewDecoder(r)
	}
	return i
}

func (i *Instance) eventLoop() {
	for {
		var ev event.Event
		if err := i.dec.Decode(&ev); err != nil {
			continue
		}

		switch ev.Name {
		case "record":
			i.eng.Emit(ev.Record)
		}
	}
}

func (i *Instance) Configure() error {
	go i.eventLoop()
	b, err := Encode(i.conf)
	if err != nil {
		return err
	}
	ev := &event.Event{Name: "config", Payload: b}
	return i.enc.Encode(ev)
}

func (i *Instance) Start() error {
	ev := &event.Event{Name: "start"}
	return i.enc.Encode(ev)
}

func (i *Instance) Emit(record *event.Record) error {
	ev := &event.Event{Name: "record", Record: record}
	return i.enc.Encode(ev)
}
