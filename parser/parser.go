package parser

type Parser interface {
	Parse(string) (map[string]interface{}, error)
}

func Get(format string) (parser Parser, err error) {
	switch format {
	case "ltsv":
		parser = &LTSVParser{}
	default:
		parser, err = NewRegexpParser(format)
	}
	return
}
