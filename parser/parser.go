package parser

import "encoding/json"

const (
	nginx     = `(?P<remote>[^ ]*) (?P<host>[^ ]*) (?P<user>[^ ]*) \[(?P<time>[^]]*)\] "(?P<method>\S+)(?: +(?P<path>[^" ]*) +\S+)?" (?P<code>\d*) (?P<size>\d*)(?: "(?P<referer>[^"]*)" "(?P<agent>[^"]*)")?.*`
	nginxTime = "02/Jan/2006:15:04:05 -0700"
)

type Parser interface {
	Parse(string) (map[string]interface{}, error)
}

type ParserFunc func(string) (map[string]interface{}, error)

func (f ParserFunc) Parse(s string) (map[string]interface{}, error) {
	return f(s)
}

var nopParser = ParserFunc(func(s string) (map[string]interface{}, error) {
	return map[string]interface{}{"message": s}, nil
})

var jsonParser = ParserFunc(func(s string) (map[string]interface{}, error) {
	var v map[string]interface{}
	err := json.Unmarshal([]byte(s), &v)
	return v, err
})

func Get(format, timeFormat, tz string) (p Parser, tp TimeParser, err error) {
	switch format {
	case "":
		p = nopParser
	case "ltsv":
		p = &LTSVParser{}
	case "json":
		p = jsonParser
	case "nginx":
		p, err = NewRegexpParser(nginx)
		timeFormat = nginxTime
	default:
		p, err = NewRegexpParser(format)
	}
	if err != nil {
		return
	}

	switch timeFormat {
	case "":
	case "unix", "unixtime":
		tp, err = NewUnixTimeParser(tz)
	default:
		tp, err = NewStrTimeParser(timeFormat, tz)
	}
	return
}

var DefaultParser = nopParser
