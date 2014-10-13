package main

import (
	"io/ioutil"

	"github.com/robertkrimen/otto"
	_ "github.com/robertkrimen/otto/underscore"
	"github.com/yosisa/fluxion/message"
	"github.com/yosisa/fluxion/plugin"
)

type Config struct {
	Script     string                 `toml:"script"`
	ScriptFile string                 `toml:"script_file"`
	Env        map[string]interface{} `toml:"env"`
}

type JSFilter struct {
	env    *plugin.Env
	conf   *Config
	vm     *otto.Otto
	script *otto.Script
}

func (f *JSFilter) Name() string {
	return "filter-js"
}

func (f *JSFilter) Init(env *plugin.Env) (err error) {
	f.env = env
	f.conf = &Config{}
	if err = env.ReadConfig(f.conf); err != nil {
		return
	}

	if f.conf.ScriptFile != "" {
		b, err := ioutil.ReadFile(f.conf.ScriptFile)
		if err != nil {
			return err
		}
		f.conf.Script = string(b)
	}

	f.vm = otto.New()
	f.script, err = f.vm.Compile("", f.conf.Script)
	return
}

func (f *JSFilter) Start() error {
	return nil
}

func (f *JSFilter) Filter(ev *message.Event) (*message.Event, error) {
	var dropped bool
	f.vm.Set("$", map[string]interface{}{
		"env": f.conf.Env,
		"event": map[string]interface{}{
			"tag":    ev.Tag,
			"time":   ev.Time,
			"record": ev.Record,
		},
		"drop": func(call otto.FunctionCall) otto.Value {
			dropped = true
			return otto.Value{}
		},
	})
	_, err := f.vm.Run(f.script)
	if err != nil {
		return nil, err
	} else if dropped {
		return nil, nil
	}
	return ev, nil
}

func main() {
	plugin.New(func() plugin.Plugin {
		return &JSFilter{}
	}).Run()
}
