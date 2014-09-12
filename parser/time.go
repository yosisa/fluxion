package parser

import "time"

type TimeParser struct {
	layout string
	loc    *time.Location
}

func NewTimeParser(layout string, tz string) (*TimeParser, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, err
	}
	return &TimeParser{layout: layout, loc: loc}, nil
}

func (p *TimeParser) Parse(s string) (t time.Time, err error) {
	t, err = time.ParseInLocation(p.layout, s, p.loc)
	if err != nil {
		return
	}
	if t.Year() == 0 {
		t = t.AddDate(time.Now().Year(), 0, 0)
	}
	return
}
