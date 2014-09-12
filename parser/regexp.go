package parser

import (
	"errors"

	"regexp"
)

type RegexpParser struct {
	re *regexp.Regexp
}

func NewRegexpParser(expr string) (*RegexpParser, error) {
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}
	return &RegexpParser{re: re}, nil
}

func (p *RegexpParser) Parse(s string) (map[string]interface{}, error) {
	r := make(map[string]interface{})
	match := p.re.FindStringSubmatch(s)
	if match == nil {
		return nil, errors.New("Not match")
	}

	for i, name := range p.re.SubexpNames() {
		if name != "" {
			r[name] = match[i]
		}
	}
	return r, nil
}
