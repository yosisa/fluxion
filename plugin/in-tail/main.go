package main

import (
	"io"
	"sync"
	"time"

	"github.com/yosisa/fluxion/event"
	"github.com/yosisa/fluxion/parser"
	"github.com/yosisa/fluxion/plugin"
	"gopkg.in/fsnotify.v1"
)

var posFiles = make(map[string]*PositionFile)

type Config struct {
	Tag          string `codec:"tag"`
	Path         string `codec:"path"`
	PosFile      string `codec:"pos_file"`
	Format       string `codec:"format"`
	TimeKey      string `codec:"time_key"`
	TimeFormat   string `codec:"time_format"`
	TimeZone     string `codec:"timezone"`
	ReadFromHead bool   `codec:"read_from_head"`
}

type TailInput struct {
	conf       Config
	parser     parser.Parser
	timeParser *parser.TimeParser
	pf         *PositionFile
}

func (i *TailInput) Init(f plugin.ConfigFeeder) (err error) {
	if err = f(&i.conf); err != nil {
		return
	}
	if i.conf.TimeKey == "" {
		i.conf.TimeKey = "time"
	}
	i.parser, i.timeParser, err = parser.Get(i.conf.Format, i.conf.TimeFormat, i.conf.TimeZone)
	if err != nil {
		return
	}

	pf, ok := posFiles[i.conf.PosFile]
	if !ok {
		if pf, err = NewPositionFile(i.conf.PosFile); err != nil {
			return
		}
		posFiles[i.conf.PosFile] = pf
	}
	i.pf = pf
	return
}

func (i *TailInput) Start() error {
	pe := i.pf.Get(i.conf.Path)
	pe.ReadFromHead = i.conf.ReadFromHead
	NewWatcher(pe, i.parseLine)
	return nil
}

func (i *TailInput) parseLine(line []byte) {
	v, err := i.parser.Parse(string(line))
	if err != nil {
		return
	}

	var record *event.Record
	if i.conf.TimeKey != "" && i.timeParser != nil {
		if s, ok := v[i.conf.TimeKey].(string); ok {
			t, err := i.timeParser.Parse(s)
			if err == nil {
				delete(v, i.conf.TimeKey)
				record = event.NewRecordWithTime(i.conf.Tag, t, v)
			}
		}
	}
	if record == nil {
		record = event.NewRecord(i.conf.Tag, v)
	}
	plugin.Emit(record)
}

type TailHandler func([]byte)

type Watcher struct {
	pe       *PositionEntry
	fsw      *fsnotify.Watcher
	r        *PositionReader
	handler  TailHandler
	rotating bool
	m        sync.Mutex
}

func NewWatcher(pe *PositionEntry, h TailHandler) *Watcher {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	w := &Watcher{
		pe:      pe,
		fsw:     fsw,
		handler: h,
	}
	w.open()
	go w.eventLoop()
	return w
}

func (w *Watcher) open() {
	w.m.Lock()
	defer w.m.Unlock()

	w.rotating = false
	if w.r != nil {
		w.r.Close()
	}

	r, err := NewPositionReader(w.pe)
	if err != nil {
		plugin.Log.Warning(err, ", wait for creation")
	} else {
		w.r = r
		w.fsw.Add(w.pe.Path)
	}
}

func (w *Watcher) eventLoop() {
	tick := time.Tick(10 * time.Second)
	for {
		select {
		case ev := <-w.fsw.Events:
			plugin.Log.Debug(ev)
			if err := w.Scan(); err != nil {
				plugin.Log.Warning(err)
			}
		case err := <-w.fsw.Errors:
			plugin.Log.Warning(err)
		case <-tick:
			if err := w.Scan(); err != nil {
				plugin.Log.Warning(err)
			}
		}
	}
}

func (w *Watcher) Scan() error {
	// To make Scan run only one thread at a time.
	// Also used to block rotation until current scanning completed.
	w.m.Lock()
	defer w.m.Unlock()

	if !w.rotating && w.pe.IsRotated() {
		plugin.Log.Infof("Rotation detected: %s", w.pe.Path)
		var wait time.Duration
		if w.r != nil {
			wait = 5 * time.Second
		}
		w.rotating = true
		time.AfterFunc(wait, w.open)
	}

	if w.r == nil {
		return nil
	}

	for {
		line, _, err := w.r.ReadLine()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		w.handler(line)
	}
	return nil
}

func main() {
	plugin.New(func() plugin.Plugin {
		return &TailInput{}
	}).Run()
}
