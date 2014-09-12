package engine

import "github.com/ugorji/go/codec"

var mh = &codec.MsgpackHandle{RawToString: true, WriteExt: true}

func Encode(v interface{}) ([]byte, error) {
	var b []byte
	err := codec.NewEncoderBytes(&b, mh).Encode(v)
	return b, err
}

func Decode(b []byte, v interface{}) error {
	return codec.NewDecoderBytes(b, mh).Decode(v)
}
