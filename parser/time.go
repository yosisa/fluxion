package parser

import (
	"errors"
	"time"
)

var (
	ErrUnsupportedValueType = errors.New("TimeParser: unsupported value type")
)

type TimeParser interface {
	Parse(interface{}) (time.Time, error)
}

type StrTimeParser struct {
	layout string
	loc    *time.Location
}

func NewStrTimeParser(layout string, tz string) (*StrTimeParser, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, err
	}
	return &StrTimeParser{layout: layout, loc: loc}, nil
}

func (p *StrTimeParser) Parse(v interface{}) (t time.Time, err error) {
	s, ok := v.(string)
	if !ok {
		err = ErrUnsupportedValueType
		return
	}
	t, err = time.ParseInLocation(p.layout, s, p.loc)
	if err != nil {
		return
	}
	if t.Year() == 0 {
		t = t.AddDate(time.Now().Year(), 0, 0)
	}
	return
}

type UnixTimeParser struct {
	loc *time.Location
}

func NewUnixTimeParser(tz string) (*UnixTimeParser, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, err
	}
	return &UnixTimeParser{loc: loc}, nil
}

func (p *UnixTimeParser) Parse(v interface{}) (t time.Time, err error) {
	n, ok := v.(float64)
	if !ok {
		err = ErrUnsupportedValueType
		return
	}
	sec := int64(n)
	nsec := int64((n - float64(sec)) * 1e+9)
	t = time.Unix(sec, nsec).In(p.loc)
	return
}
