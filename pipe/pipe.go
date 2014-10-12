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
	r   io.Reader
	w   io.Writer
	enc event.Encoder
	dec event.Decoder
	rm  sync.Mutex
	wm  sync.Mutex
}

func NewInterProcess(r io.Reader, w io.Writer) *InterProcess {
	p := &InterProcess{}
	if r != nil {
		p.r = r
		p.dec = event.NewDecoder(r)
	}
	if w != nil {
		p.w = w
		p.enc = event.NewEncoder(w)
	}
	return p
}

func (p *InterProcess) Read() (*message.Message, error) {
	p.rm.Lock()
	defer p.rm.Unlock()
	b := make([]byte, 1)
	if _, err := p.r.Read(b); err != nil {
		return nil, err
	}
	m := &message.Message{Type: message.MessageType(b[0])}
	m.Decode(p.dec)
	return m, nil
}

func (p *InterProcess) Write(m *message.Message) error {
	p.wm.Lock()
	defer p.wm.Unlock()
	if _, err := p.w.Write([]byte{byte(m.Type)}); err != nil {
		return err
	}
	return m.Encode(p.enc)
}
