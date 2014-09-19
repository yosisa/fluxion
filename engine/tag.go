package engine

import (
	"regexp"
	"strings"

	"github.com/yosisa/fluxion/event"
)

type Emitter interface {
	Emit(*event.Record) error
}

type TagRouter struct {
	patterns []*regexp.Regexp
	values   []Emitter
}

func (t *TagRouter) Add(pattern string, e Emitter) error {
	re, err := compileTag(pattern)
	if err != nil {
		return err
	}
	t.patterns = append(t.patterns, re)
	t.values = append(t.values, e)
	return nil
}

func (t *TagRouter) Route(tag string) Emitter {
	for i, re := range t.patterns {
		if re.MatchString(tag) {
			return t.values[i]
		}
	}
	return nil
}

func compileTag(s string) (*regexp.Regexp, error) {
	if s == "**" || s == "*" {
		s = `.*`
	} else {
		s = strings.Replace(s, `.`, `\.`, -1)
		s = strings.Replace(s, `**`, `(\..+|)`, -1)
		s = strings.Replace(s, `\.(\..+|)`, `(\..+|)`, -1)
		s = strings.Replace(s, `*`, `.*`, -1)
	}
	s = `^` + s + `$`
	return regexp.Compile(s)
}
