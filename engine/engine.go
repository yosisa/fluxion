package engine

import (
	"fmt"
	"log"

	"time"

	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/pave/process"
)

type Engine struct {
	pm      *process.ProcessManager
	plugins []*Instance
	tr      *TagRouter
	bufs    map[string]*buffer.Options
}

func New() *Engine {
	return &Engine{
		pm: process.NewProcessManager(process.StrategyRestartAlways, 3*time.Second),
		tr: &TagRouter{},
		bufs: map[string]*buffer.Options{
			"default": buffer.DefaultOptions,
		},
	}
}

func (e *Engine) RegisterBuffer(opts *buffer.Options) {
	e.bufs[opts.Name] = opts
}

func (e *Engine) RegisterInputPlugin(conf map[string]interface{}) {
	command := "fluxion-in-" + conf["type"].(string)
	cmd := process.NewCommand(command)
	e.plugins = append(e.plugins, NewInstance(e, cmd, conf, nil))
	e.pm.Add(cmd)
}

func (e *Engine) RegisterOutputPlugin(conf map[string]interface{}) error {
	bufName := "default"
	if name, ok := conf["buffer_name"].(string); ok {
		bufName = name
	}
	buf, ok := e.bufs[bufName]
	if !ok {
		return fmt.Errorf("No such buffer defined: %s", bufName)
	}

	command := "fluxion-out-" + conf["type"].(string)
	cmd := process.NewCommand(command)
	ins := NewInstance(e, cmd, conf, buf)
	e.plugins = append(e.plugins, ins)
	if err := e.tr.Add(conf["match"].(string), ins); err != nil {
		log.Fatal(err)
	}
	e.pm.Add(cmd)
	return nil
}

func (e *Engine) Emit(record *event.Record) {
	if ins := e.tr.Route(record.Tag); ins != nil {
		ins.Emit(record)
	} else {
		fmt.Printf("No output plugin exists for tag %s, discard.\n", record.Tag)
	}
}

func (e *Engine) Start() {
	e.pm.Start()
	for _, p := range e.plugins {
		p.SetBuffer()
	}
	for _, p := range e.plugins {
		p.Configure()
	}
	for _, p := range e.plugins {
		p.Start()
	}
}

func (e *Engine) Wait() {
	e.pm.Wait()
}
