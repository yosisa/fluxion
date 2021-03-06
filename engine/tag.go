package engine

import (
	"regexp"
	"strings"

	"github.com/yosisa/fluxion/message"
)

type Emitter interface {
	Emit(*message.Event) error
}

type TagRouter struct {
	patterns []*regexp.Regexp
	values   []Emitter
}

func (t *TagRouter) Add(re *regexp.Regexp, e Emitter) {
	t.patterns = append(t.patterns, re)
	t.values = append(t.values, e)
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
