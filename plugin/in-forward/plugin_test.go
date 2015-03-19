package in_forward

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"github.com/ugorji/go/codec"
	"github.com/yosisa/fluxion/message"
	"github.com/yosisa/fluxion/plugin"
	. "gopkg.in/check.v1"
)

type rwBuffer struct {
	r *bytes.Buffer
	w *bytes.Buffer
}

func newRWBuffer() *rwBuffer {
	return &rwBuffer{
		r: new(bytes.Buffer),
		w: new(bytes.Buffer),
	}
}

func (rw *rwBuffer) Read(b []byte) (int, error) {
	return rw.r.Read(b)
}

func (rw *rwBuffer) Write(b []byte) (int, error) {
	return rw.w.Write(b)
}

func Test(t *testing.T) { TestingT(t) }

type HandleConnection struct {
	events []*message.Event
	buf    *rwBuffer
	enc    *codec.Encoder
	p      *ForwardInput
}

var _ = Suite(&HandleConnection{})

func (s *HandleConnection) SetUpTest(c *C) {
	s.events = nil
	s.buf = newRWBuffer()
	s.enc = codec.NewEncoder(s.buf.r, mh)
	s.p = &ForwardInput{
		env: &plugin.Env{
			Emit: func(ev *message.Event) {
				s.events = append(s.events, ev)
			},
		},
	}
}

func (s *HandleConnection) TestFlatEncoding(c *C) {
	t := time.Now()
	s.enc.Encode([]interface{}{"flat", t, map[string]interface{}{"key": "value"}})
	s.p.handleConnection(s.buf)
	c.Assert(len(s.events), Equals, 1)
	ev := s.events[0]
	c.Assert(ev.Tag, Equals, "flat")
	c.Assert(ev.Time, Equals, t)
	c.Assert(ev.Record, DeepEquals, map[string]interface{}{"key": "value"})
}

func (s *HandleConnection) TestNestedEncoding(c *C) {
	t1 := time.Now()
	t2 := t1.Add(time.Second)
	b := new(bytes.Buffer)
	enc := codec.NewEncoder(b, mh)
	enc.Encode([]interface{}{t1, map[string]interface{}{"seq": 1}})
	enc.Encode([]interface{}{t2, map[string]interface{}{"seq": 2}})
	s.enc.Encode([]interface{}{"nested", b.Bytes()})
	s.p.handleConnection(s.buf)
	c.Assert(len(s.events), Equals, 2)

	ev := s.events[0]
	c.Assert(ev.Tag, Equals, "nested")
	c.Assert(ev.Time, Equals, t1)
	c.Assert(ev.Record, DeepEquals, map[string]interface{}{"seq": int64(1)})
	ev = s.events[1]
	c.Assert(ev.Tag, Equals, "nested")
	c.Assert(ev.Time, Equals, t2)
	c.Assert(ev.Record, DeepEquals, map[string]interface{}{"seq": int64(2)})
}

func (s *HandleConnection) TestExtendedFlatEncoding(c *C) {
	t := time.Now()
	opts := map[string]interface{}{"chunk": 1}
	s.enc.Encode([]interface{}{"flat-ex", t, map[string]interface{}{"key": "value"}, opts})
	s.p.handleConnection(s.buf)
	c.Assert(len(s.events), Equals, 1)
	ev := s.events[0]
	c.Assert(ev.Tag, Equals, "flat-ex")
	c.Assert(ev.Time, Equals, t)
	c.Assert(ev.Record, DeepEquals, map[string]interface{}{"key": "value"})
	resp, _ := ioutil.ReadAll(s.buf.w)
	c.Assert(resp, DeepEquals, []byte{0x81, 0xa3, 'a', 'c', 'k', 0x01})
}

func (s *HandleConnection) TestExtendedNestedEncoding(c *C) {
	t1 := time.Now()
	t2 := t1.Add(time.Second)
	b := new(bytes.Buffer)
	enc := codec.NewEncoder(b, mh)
	enc.Encode([]interface{}{t1, map[string]interface{}{"seq": 1}})
	enc.Encode([]interface{}{t2, map[string]interface{}{"seq": 2}})
	opts := map[string]interface{}{"chunk": 1}
	s.enc.Encode([]interface{}{"nested-ex", b.Bytes(), opts})
	s.p.handleConnection(s.buf)
	c.Assert(len(s.events), Equals, 2)

	ev := s.events[0]
	c.Assert(ev.Tag, Equals, "nested-ex")
	c.Assert(ev.Time, Equals, t1)
	c.Assert(ev.Record, DeepEquals, map[string]interface{}{"seq": int64(1)})
	ev = s.events[1]
	c.Assert(ev.Tag, Equals, "nested-ex")
	c.Assert(ev.Time, Equals, t2)
	c.Assert(ev.Record, DeepEquals, map[string]interface{}{"seq": int64(2)})
	resp, _ := ioutil.ReadAll(s.buf.w)
	c.Assert(resp, DeepEquals, []byte{0x81, 0xa3, 'a', 'c', 'k', 0x01})
}

type Time struct{}

var _ = Suite(&Time{})

func (s *Time) TestParseTime(c *C) {
	now := time.Now()
	b, err := now.MarshalBinary()
	c.Assert(err, IsNil)

	tt, err := parseTime(b)
	c.Assert(err, IsNil)
	c.Assert(tt, Equals, now)

	tt, err = parseTime(string(b))
	c.Assert(err, IsNil)
	c.Assert(tt, Equals, now)
}

func (s *Time) TestParseCompatibleTime(c *C) {
	epoch := time.Now().Unix()
	now := time.Unix(epoch, 0)

	tt, err := parseTime(int64(epoch))
	c.Assert(err, IsNil)
	c.Assert(tt, Equals, now)

	tt, err = parseTime(uint64(epoch))
	c.Assert(err, IsNil)
	c.Assert(tt, Equals, now)

	tt, err = parseTime(float64(epoch))
	c.Assert(err, IsNil)
	c.Assert(tt, Equals, now)

	tt, err = parseTime(uint32(epoch))
	c.Assert(err, IsNil)
	c.Assert(tt, Equals, now)
}
