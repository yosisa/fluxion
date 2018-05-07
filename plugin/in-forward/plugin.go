package in_forward

import (
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/ugorji/go/codec"
	"github.com/yosisa/fluxion/message"
	"github.com/yosisa/fluxion/plugin"
)

var mh = &codec.MsgpackHandle{RawToString: true}

type Config struct {
	Bind string `toml:"bind"`
}

type ForwardInput struct {
	env     *plugin.Env
	conf    *Config
	ln      net.Listener
	udpConn *net.UDPConn
	closed  bool
}

func (i *ForwardInput) Init(env *plugin.Env) error {
	i.env = env
	i.conf = &Config{}
	return env.ReadConfig(i.conf)
}

func (i *ForwardInput) Start() error {
	ln, err := net.Listen("tcp", i.conf.Bind)
	if err != nil {
		return err
	}
	addr, err := net.ResolveUDPAddr("udp", i.conf.Bind)
	if err != nil {
		ln.Close()
		return err
	}
	sock, err := net.ListenUDP("udp", addr)
	if err != nil {
		ln.Close()
		return err
	}

	i.ln = ln
	i.udpConn = sock
	go i.accept()
	go i.heartbeatHandler()
	return nil
}

func (i *ForwardInput) Close() error {
	i.closed = true
	err1 := i.ln.Close()
	err2 := i.udpConn.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func (i *ForwardInput) accept() {
	for !i.closed {
		conn, err := i.ln.Accept()
		if err != nil {
			continue
		}
		go i.handleConnection(conn)
	}
}

func (i *ForwardInput) handleConnection(rw io.ReadWriter) {
	dec := codec.NewDecoder(rw, mh)
	for {
		var v []interface{}
		err := dec.Decode(&v)
		if err != nil {
			if err != io.EOF {
				i.env.Log.Warning(err)
			}
			return
		}

		tag := v[0].(string)
		switch len(v) {
		case 2:
			i.parseNestedEncoding(tag, v)
		case 3:
			if isTimeFormat(v[1]) {
				i.parseFlatEncoding(tag, v)
			} else {
				i.parseNestedEncoding(tag, v)
				i.handleOption(rw, v[2].(map[string]interface{}))
			}
		case 4:
			i.parseFlatEncoding(tag, v)
			i.handleOption(rw, v[3].(map[string]interface{}))
		}
	}
}

func (i *ForwardInput) parseNestedEncoding(tag string, v []interface{}) {
	var dec *codec.Decoder
	switch typed := v[1].(type) {
	case []byte:
		dec = codec.NewDecoderBytes(typed, mh)
	case string:
		dec = codec.NewDecoderBytes([]byte(typed), mh)
	}

	for {
		var v2 []interface{}
		if err := dec.Decode(&v2); err != nil {
			if err != io.EOF {
				i.env.Log.Warning(err)
			}
			return
		}

		t, err := parseTime(v2[0])
		if err != nil {
			i.env.Log.Errorf("Time decode error: %v, skipping", err)
			continue
		}
		r := message.NewEventWithTime(tag, t, v2[1].(map[string]interface{}))
		i.env.Emit(r)
	}
}

func (i *ForwardInput) parseFlatEncoding(tag string, v []interface{}) {
	t, err := parseTime(v[1])
	if err != nil {
		i.env.Log.Errorf("Time decode error: %v, skipping", err)
		return
	}
	r := message.NewEventWithTime(tag, t, v[2].(map[string]interface{}))
	i.env.Emit(r)
}

func (i *ForwardInput) handleOption(w io.Writer, opts map[string]interface{}) {
	if id, ok := opts["chunk"]; ok {
		resp := map[string]interface{}{"ack": id}
		var b []byte
		if err := codec.NewEncoderBytes(&b, mh).Encode(resp); err != nil {
			i.env.Log.Errorf("Failed to encode response: %v", err)
		}
		if _, err := w.Write(b); err != nil {
			i.env.Log.Errorf("Failed to respond ack: chunk id %v, err: %v", id, err)
		}
	}
}

func (i *ForwardInput) heartbeatHandler() {
	buf := make([]byte, 1024)
	res := []byte{0}
	for {
		_, remote, err := i.udpConn.ReadFromUDP(buf)
		if i.closed {
			return
		}
		if err != nil {
			i.env.Log.Warning(err)
			continue
		}
		i.udpConn.WriteToUDP(res, remote)
	}
}

func isTimeFormat(v interface{}) bool {
	switch typed := v.(type) {
	case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case time.Time, *time.Time:
		return true
	case []byte:
		return len(typed) == 15 && typed[0] == 1
	case string:
		switch len(typed) {
		case 4, 8, 12:
			return true
		case 15:
			// 1 is timeBinaryVersion defined in time.go.
			// In msgpack spec, 0x01 means positive fixint, thus
			// it's never appeared in this context.
			return typed[0] == 1
		}
	}
	return false
}

func parseTime(v interface{}) (t time.Time, err error) {
	var n int64
	switch typed := v.(type) {
	case []byte:
		err = t.UnmarshalBinary(typed)
		return
	case string:
		if len(typed) == 15 && typed[0] == 1 {
			// Binary format in Time.MarshalBinary()
			t = t.UTC()
			err = t.UnmarshalBinary([]byte(typed))
		} else if len(typed) == 4 {
			// MessagePack timestamp extension type (32-bit)
			err = codec.NewDecoderBytes(append([]byte{0xd6, 0xff}, []byte(typed)...), mh).Decode(&t)
		} else if len(typed) == 8 {
			// MessagePack timestamp extension type (64-bit)
			err = codec.NewDecoderBytes(append([]byte{0xd7, 0xff}, []byte(typed)...), mh).Decode(&t)
		} else if len(typed) == 12 {
			// MessagePack timestamp extension type (96-bit)
			err = codec.NewDecoderBytes(append([]byte{0xc7, 0x0c, 0xff}, []byte(typed)...), mh).Decode(&t)
		} else {
			err = fmt.Errorf("unknown time string format [%s]", hex.EncodeToString([]byte(typed)))
		}
		return
	case int64:
		n = typed
	case uint64:
		n = int64(typed)
	case float64:
		nsec := int64((typed - float64(int64(typed))) * 1e9)
		t = time.Unix(int64(typed), nsec)
		return
	case time.Time:
		t = typed
		return
	case *time.Time:
		t = *typed
		return
	default:
		if n, err = strconv.ParseInt(fmt.Sprintf("%v", typed), 10, 64); err != nil {
			return
		}
	}
	t = time.Unix(n, 0)
	return
}

func Factory() plugin.Plugin {
	return &ForwardInput{}
}

func init() {
	mh.MapType = reflect.TypeOf(map[string]interface{}(nil))
}
