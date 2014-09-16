package main

import (
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/ugorji/go/codec"
	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/fluxion/plugin"
)

var mh = &codec.MsgpackHandle{}

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

		tag := string(v[0].([]byte))
		switch len(v) {
		case 2:
			dec2 := codec.NewDecoderBytes(v[1].([]byte), mh)
			for {
				var v2 []interface{}
				if err := dec2.Decode(&v2); err != nil {
					if err != io.EOF {
						log.Print(err)
					}
					break
				}

				t := time.Unix(int64(v2[0].(uint64)), 0)
				r := event.NewRecordWithTime(tag, t, v2[1])
				plugin.Emit(r)
			}
		case 3:
			t := time.Unix(int64(v[1].(uint64)), 0)
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

func main() {
	log.SetOutput(os.Stderr)
	plugin.New(&ForwardInput{}).Run()
}
