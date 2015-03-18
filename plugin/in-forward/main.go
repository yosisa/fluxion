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

func (i *ForwardInput) handleConnection(r io.Reader) {
	dec := codec.NewDecoder(r, mh)
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
			dec2 := codec.NewDecoderBytes([]byte(v[1].(string)), mh)
			for {
				var v2 []interface{}
				if err := dec2.Decode(&v2); err != nil {
					if err != io.EOF {
						i.env.Log.Warning(err)
					}
					break
				}

				t, err := parseTime(v2[0])
				if err != nil {
					i.env.Log.Errorf("Time decode error: %v, skipping", err)
					continue
				}
				r := message.NewEventWithTime(tag, t, v2[1].(map[string]interface{}))
				i.env.Emit(r)
			}
		case 3:
			t, err := parseTime(v[1])
			if err != nil {
				i.env.Log.Errorf("Time decode error: %v, skipping", err)
				continue
			}
			r := message.NewEventWithTime(tag, t, v[2].(map[string]interface{}))
			i.env.Emit(r)
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
