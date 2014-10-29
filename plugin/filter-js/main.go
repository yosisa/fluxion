package main

import (
	"io/ioutil"
	"time"

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
		"emit": func(call otto.FunctionCall) (ret otto.Value) {
			var record otto.Value
			var t time.Time
			tag := call.Argument(0).String()
			if !call.Argument(2).IsDefined() {
				record = call.Argument(1)
				t = time.Now()
			} else {
				record = call.Argument(2)
				v, err := call.Argument(1).Export()
				if err != nil {
					f.env.Log.Warningf("Failed to get time: %v", err)
					return
				}
				var ok bool
				t, ok = v.(time.Time)
				if !ok {
					f.env.Log.Warningf("Failed to get time: unsupported type %T", v)
					return
				}
			}
			rec, err := record.Export()
			if err != nil {
				f.env.Log.Warningf("Failed to get record: %v", err)
				return
			}
			typedRec, ok := rec.(map[string]interface{})
			if !ok {
				f.env.Log.Warningf("Failed to get record: Unsupported type %T", rec)
				return
			}
			f.env.Emit(message.NewEventWithTime(tag, t, typedRec))
			return
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

func (f *JSFilter) Close() error {
	return nil
}

func main() {
	plugin.New("filter-js", func() plugin.Plugin {
		return &JSFilter{}
	}).Run()
}
