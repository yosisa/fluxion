package pipe

import (
	"io"

	"github.com/yosisa/fluxion/event"
)

type Pipe struct {
	R <-chan *event.Event
	W chan<- *event.Event
}

func NewInProcess() *Pipe {
	ch := make(chan *event.Event, 100)
	return &Pipe{R: ch, W: ch}
}

func NewInterProcess(r io.Reader, w io.Writer) *Pipe {
	p := &Pipe{}
	if r != nil {
		dec := event.NewDecoder(r)
		ch := make(chan *event.Event, 100)
		go readEvent(dec, ch)
		p.R = ch
	}

	if w != nil {
		enc := event.NewEncoder(w)
		ch := make(chan *event.Event)
		go writeEvent(enc, ch)
		p.W = ch
	}
	return p
}

func readEvent(d event.Decoder, c chan *event.Event) {
	for {
		var ev event.Event
		if err := d.Decode(&ev); err != nil {
			close(c)
			return
		}

		c <- &ev
	}
}

func writeEvent(e event.Encoder, c chan *event.Event) {
	for ev := range c {
		if err := e.Encode(ev); err != nil {
			close(c)
		}
	}
}
