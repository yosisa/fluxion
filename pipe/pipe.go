package pipe

import (
	"io"

	"github.com/yosisa/fluxion/event"
)

type Pipe interface {
	Read() (*event.Event, error)
	Write(*event.Event) error
}

type InProcess struct {
	c chan *event.Event
}

func NewInProcess() *InProcess {
	return &InProcess{c: make(chan *event.Event, 100)}
}

func (p *InProcess) Read() (*event.Event, error) {
	ev := <-p.c
	return ev, nil
}

func (p *InProcess) Write(ev *event.Event) error {
	p.c <- ev
	return nil
}

type InterProcess struct {
	enc event.Encoder
	dec event.Decoder
}

func NewInterProcess(r io.Reader, w io.Writer) *InterProcess {
	p := &InterProcess{}
	if r != nil {
		p.dec = event.NewDecoder(r)
	}
	if w != nil {
		p.enc = event.NewEncoder(w)
	}
	return p
}

func (p *InterProcess) Read() (*event.Event, error) {
	var ev event.Event
	if err := p.dec.Decode(&ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

func (p *InterProcess) Write(ev *event.Event) error {
	return p.enc.Encode(ev)
}
