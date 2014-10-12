package pipe

import (
	"io"
	"sync"

	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/fluxion/message"
)

type Pipe interface {
	Read() (*message.Message, error)
	Write(*message.Message) error
}

type InProcess struct {
	c chan *message.Message
}

func NewInProcess() *InProcess {
	return &InProcess{c: make(chan *message.Message, 100)}
}

func (p *InProcess) Read() (*message.Message, error) {
	ev := <-p.c
	return ev, nil
}

func (p *InProcess) Write(ev *message.Message) error {
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

func (p *InterProcess) Read() (*message.Message, error) {
	p.rm.Lock()
	defer p.rm.Unlock()
	return message.Decode(p.dec)
}

func (p *InterProcess) Write(m *message.Message) error {
	p.wm.Lock()
	defer p.wm.Unlock()
	return message.Encode(p.enc, m)
}
