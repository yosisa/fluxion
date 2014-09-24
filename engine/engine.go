package engine

import (
	"fmt"
	"log"
	"sync/atomic"

	"time"

	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/pave/process"
)

type Engine struct {
	pm      *process.ProcessManager
	plugins map[string]*Instance
	units   []*ExecUnit
	filters []*ExecUnit
	tr      map[string]*TagRouter
	ftr     *TagRouter
	bufs    map[string]*buffer.Options
	unitID  int32
}

func New() *Engine {
	return &Engine{
		pm:      process.NewProcessManager(process.StrategyRestartAlways, 3*time.Second),
		plugins: make(map[string]*Instance),
		tr:      make(map[string]*TagRouter),
		ftr:     &TagRouter{},
		bufs: map[string]*buffer.Options{
			"default": buffer.DefaultOptions,
		},
	}
}

func (e *Engine) RegisterBuffer(opts *buffer.Options) {
	e.bufs[opts.Name] = opts
}

func (e *Engine) pluginInstance(command string) *Instance {
	if ins, ok := e.plugins[command]; ok {
		return ins
	}
	cmd := process.NewCommand(command)
	e.pm.Add(cmd)
	ins := NewInstance(e, cmd)
	e.plugins[command] = ins
	return ins
}

func (e *Engine) addExecUnit(ins *Instance, conf map[string]interface{}, bopts *buffer.Options) *ExecUnit {
	unit := ins.AddExecUnit(atomic.AddInt32(&e.unitID, 1), conf, bopts)
	e.units = append(e.units, unit)
	return unit
}

func (e *Engine) RegisterInputPlugin(conf map[string]interface{}) {
	ins := e.pluginInstance("fluxion-in-" + conf["type"].(string))
	e.addExecUnit(ins, conf, nil)
}

func (e *Engine) RegisterOutputPlugin(name string, conf map[string]interface{}) error {
	bufName := "default"
	if name, ok := conf["buffer_name"].(string); ok {
		bufName = name
	}
	buf, ok := e.bufs[bufName]
	if !ok {
		return fmt.Errorf("No such buffer defined: %s", bufName)
	}

	ins := e.pluginInstance("fluxion-out-" + conf["type"].(string))
	unit := e.addExecUnit(ins, conf, buf)

	tr, ok := e.tr[name]
	if !ok {
		tr = &TagRouter{}
		e.tr[name] = tr
	}
	if err := tr.Add(conf["match"].(string), unit); err != nil {
		log.Fatal(err)
	}
	return nil
}

func (e *Engine) RegisterFilterPlugin(conf map[string]interface{}) {
	ins := e.pluginInstance("fluxion-filter-" + conf["type"].(string))
	unit := e.addExecUnit(ins, conf, nil)

	pattern := conf["match"].(string)
	if err := e.ftr.Add(pattern, unit); err != nil {
		log.Fatal(err)
	}

	// Register new filter to the preceding filters
	for _, f := range e.filters {
		if err := f.Router.Add(pattern, unit); err != nil {
			log.Fatal(err)
		}
	}
	e.filters = append(e.filters, unit)
}

func (e *Engine) Filter(record *event.Record) {
	if ins := e.ftr.Route(record.Tag); ins != nil {
		ins.Emit(record)
	} else {
		e.Emit(record)
	}
}

func (e *Engine) Emit(record *event.Record) {
	for _, tr := range e.tr {
		if ins := tr.Route(record.Tag); ins != nil {
			ins.Emit(record)
		} else {
			fmt.Printf("No output plugin exists for tag %s, discard.\n", record.Tag)
		}
	}
}

func (e *Engine) Start() {
	e.pm.Start()
	for _, u := range e.units {
		u.SetBuffer()
	}
	for _, u := range e.units {
		u.Configure()
	}
	for _, u := range e.units {
		u.Start()
	}
	for _, p := range e.plugins {
		p.Start()
	}
}

func (e *Engine) Wait() {
	e.pm.Wait()
}
