package parser

import "strings"

type LTSVParser struct{}

func (p *LTSVParser) Parse(s string) (map[string]interface{}, error) {
	r := make(map[string]interface{})
	pairs := strings.Split(s, "\t")
	for _, item := range pairs {
		pair := strings.SplitN(item, ":", 2)
		if len(pair) != 2 {
			continue
		}
		r[pair[0]] = pair[1]
	}
	return r, nil
}
