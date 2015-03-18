package main

import (
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
			// 1 is timeBinaryVersion defined in time.go.
			// In msgpack spec, 0x01 means positive fixint, thus
			// it's never appeared in this context.
			if s, ok := v[1].(string); ok && s[0] != 1 {
				i.parseNestedEncoding(tag, v)
				i.handleOption(rw, v[2].(map[string]interface{}))
			} else {
				i.parseFlatEncoding(tag, v)
			}
		case 4:
			i.parseFlatEncoding(tag, v)
			i.handleOption(rw, v[3].(map[string]interface{}))
		}
	}
}

func (i *ForwardInput) parseNestedEncoding(tag string, v []interface{}) {
	dec := codec.NewDecoderBytes([]byte(v[1].(string)), mh)
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

func parseTime(v interface{}) (t time.Time, err error) {
	var n int64
	switch typed := v.(type) {
	case []byte:
		err = t.UnmarshalBinary(typed)
		return
	case string:
		err = t.UnmarshalBinary([]byte(typed))
		return
	case int64:
		n = typed
	case uint64:
		n = int64(typed)
	case float64:
		nsec := int64((typed - float64(int64(typed))) * 1e9)
		t = time.Unix(int64(typed), nsec)
		return
	default:
		if n, err = strconv.ParseInt(fmt.Sprintf("%v", typed), 10, 64); err != nil {
			return
		}
	}
	t = time.Unix(n, 0)
	return
}

func main() {
	plugin.New("in-forward", func() plugin.Plugin {
		return &ForwardInput{}
	}).Run()
}

func init() {
	mh.MapType = reflect.TypeOf(map[string]interface{}(nil))
}
