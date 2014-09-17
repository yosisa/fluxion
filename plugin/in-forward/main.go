package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/ugorji/go/codec"
	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/fluxion/plugin"
)

var mh = &codec.MsgpackHandle{RawToString: true}

type Config struct {
	Bind string `codec:"bind"`
}

type ForwardInput struct {
	conf    *Config
	ln      net.Listener
	udpConn *net.UDPConn
}

func (i *ForwardInput) Init(f plugin.ConfigFeeder) error {
	i.conf = &Config{}
	return f(i.conf)
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

func (i *ForwardInput) accept() {
	for {
		conn, err := i.ln.Accept()
		if err != nil {
			continue
		}
		go i.handleConnection(conn)
	}
}

func (i *ForwardInput) handleConnection(conn net.Conn) {
	dec := codec.NewDecoder(conn, mh)
	for {
		var v []interface{}
		err := dec.Decode(&v)
		if err != nil {
			if err != io.EOF {
				log.Print(err)
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
						log.Print(err)
					}
					break
				}

				t, err := parseTime(v2[0])
				if err != nil {
					continue
				}
				r := event.NewRecordWithTime(tag, t, v2[1])
				plugin.Emit(r)
			}
		case 3:
			t, err := parseTime(v[1])
			if err != nil {
				continue
			}
			r := event.NewRecordWithTime(tag, t, v[2])
			plugin.Emit(r)
		}
	}
}

func (i *ForwardInput) heartbeatHandler() {
	buf := make([]byte, 1024)
	res := []byte{0}
	for {
		_, remote, err := i.udpConn.ReadFromUDP(buf)
		if err != nil {
			log.Fatal(err)
		}
		i.udpConn.WriteToUDP(res, remote)
	}
}

func parseTime(v interface{}) (t time.Time, err error) {
	var n int64
	switch typed := v.(type) {
	case int64:
		n = typed
	case uint64:
		n = int64(typed)
	case float64:
		n = int64(typed)
	default:
		if n, err = strconv.ParseInt(fmt.Sprintf("%v", typed), 10, 64); err != nil {
			return
		}
	}
	t = time.Unix(n, 0)
	return
}

func main() {
	log.SetOutput(os.Stderr)
	plugin.New(&ForwardInput{}).Run()
}
