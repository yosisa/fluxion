package pipe

import (
	"io"
	"sync"

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
	rm  sync.Mutex
	wm  sync.Mutex
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
	p.rm.Lock()
	defer p.rm.Unlock()

	var ev event.Event
	if err := p.dec.Decode(&ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

func (p *InterProcess) Write(ev *event.Event) error {
	p.wm.Lock()
	defer p.wm.Unlock()
	return p.enc.Encode(ev)
}
