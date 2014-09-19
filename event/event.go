package event

import (
	"io"

	"github.com/ugorji/go/codec"
	"github.com/yosisa/fluxion/buffer"
)

var mh = &codec.MsgpackHandle{RawToString: true, WriteExt: true}

type Event struct {
	UnitID  int32           `codec:"unit_id"`
	Name    string          `codec:"name"`
	Record  *Record         `codec:"record,omitempty"`
	Buffer  *buffer.Options `codec:"buffer,omitempty"`
	Payload interface{}     `codec:"payload,omitempty"`
}

type Encoder interface {
	Encode(interface{}) error
}

type Decoder interface {
	Decode(interface{}) error
}

func NewEncoder(w io.Writer) Encoder {
	return codec.NewEncoder(w, mh)
}

func NewDecoder(r io.Reader) Decoder {
	return codec.NewDecoder(r, mh)
}
