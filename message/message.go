package message

import (
	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/event"
)

type MessageType byte

const (
	TypInfoRequest MessageType = iota
	TypInfoResponse
	TypBufferOption
	TypConfigure
	TypStart
	TypStop
	TypTerminated
	TypEvent
	TypEventChain
)

type Message struct {
	Type    MessageType
	UnitID  int32
	Payload interface{}
}

type Encoder interface {
	Encode(interface{}) error
}

type Decoder interface {
	Decode(interface{}) error
}

func Encode(enc Encoder, m *Message) error {
	if err := enc.Encode(m.Type); err != nil {
		return err
	}
	if err := enc.Encode(m.UnitID); err != nil {
		return err
	}
	switch m.Type {
	case TypInfoRequest, TypStart, TypStop, TypTerminated:
		return nil
	default:
		return enc.Encode(m.Payload)
	}
}

func Decode(dec Decoder) (*Message, error) {
	var m Message
	if err := dec.Decode(&m.Type); err != nil {
		return nil, err
	}
	if err := dec.Decode(&m.UnitID); err != nil {
		return nil, err
	}

	var err error
	switch m.Type {
	case TypInfoRequest, TypStart, TypStop, TypTerminated:
		return &m, nil
	case TypInfoResponse:
	case TypBufferOption:
		var opts buffer.Options
		err = dec.Decode(&opts)
		m.Payload = &opts
	case TypConfigure:
		var b []byte
		err = dec.Decode(&b)
		m.Payload = b
	case TypEvent, TypEventChain:
		var event event.Record
		err = dec.Decode(&event)
		m.Payload = &event
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
