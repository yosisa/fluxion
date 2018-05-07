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
	t := time.Now().UTC()
	s.enc.Encode([]interface{}{"flat", t, map[string]interface{}{"key": "value"}})
	s.p.handleConnection(s.buf)
	c.Assert(len(s.events), Equals, 1)
	ev := s.events[0]
	c.Assert(ev.Tag, Equals, "flat")
	c.Assert(ev.Time, Equals, t)
	c.Assert(ev.Record, DeepEquals, map[string]interface{}{"key": "value"})
}

func (s *HandleConnection) TestNestedEncoding(c *C) {
	t1 := time.Now().UTC()
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
	t := time.Now().UTC()
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
	t1 := time.Now().UTC()
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

	c.Assert(isTimeFormat(b), Equals, true)
	tt, err := parseTime(b)
	c.Assert(err, IsNil)
	c.Assert(tt, Equals, now)

	c.Assert(isTimeFormat(string(b)), Equals, true)
	tt, err = parseTime(string(b))
	c.Assert(err, IsNil)
	c.Assert(tt, Equals, now)
}

func (s *Time) TestParseCompatibleTime(c *C) {
	epoch := time.Now().Unix()
	now := time.Unix(epoch, 0)

	c.Assert(isTimeFormat(int64(epoch)), Equals, true)
	tt, err := parseTime(int64(epoch))
	c.Assert(err, IsNil)
	c.Assert(tt, Equals, now)

	c.Assert(isTimeFormat(uint64(epoch)), Equals, true)
	tt, err = parseTime(uint64(epoch))
	c.Assert(err, IsNil)
	c.Assert(tt, Equals, now)

	c.Assert(isTimeFormat(float64(epoch)), Equals, true)
	tt, err = parseTime(float64(epoch))
	c.Assert(err, IsNil)
	c.Assert(tt, Equals, now)

	c.Assert(isTimeFormat(uint32(epoch)), Equals, true)
	tt, err = parseTime(uint32(epoch))
	c.Assert(err, IsNil)
	c.Assert(tt, Equals, now)
}

func (s *Time) TestParseMsgpackTime(c *C) {
	var b bytes.Buffer
	h := &codec.MsgpackHandle{RawToString: true}

	t32, err := time.Parse("2006/01/02 15:04", "1970/01/01 00:00")
	c.Assert(err, IsNil)
	t64, err := time.Parse("2006/01/02 15:04", "2106/02/08 00:00")
	c.Assert(err, IsNil)
	t96, err := time.Parse("2006/01/02 15:04", "2514/05/31 00:00")
	c.Assert(err, IsNil)

	err = codec.NewEncoder(&b, h).Encode(t32)
	c.Assert(err, IsNil)
	s32 := string(b.Bytes()[1:])
	c.Assert(isTimeFormat(s32), Equals, true)
	tt32, err := parseTime(s32)
	c.Assert(err, IsNil)
	c.Assert(tt32, Equals, t32)

	b.Reset()
	err = codec.NewEncoder(&b, h).Encode(t64)
	c.Assert(err, IsNil)
	s64 := string(b.Bytes()[1:])
	c.Assert(isTimeFormat(s64), Equals, true)
	tt64, err := parseTime(s64)
	c.Assert(err, IsNil)
	c.Assert(tt64, Equals, t64)

	b.Reset()
	err = codec.NewEncoder(&b, h).Encode(t96)
	c.Assert(err, IsNil)
	s96 := string(b.Bytes()[1:])
	c.Assert(isTimeFormat(s96), Equals, true)
	tt96, err := parseTime(s96)
	c.Assert(err, IsNil)
	c.Assert(tt96, Equals, t96)
}
