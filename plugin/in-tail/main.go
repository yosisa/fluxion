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
	r          *PositionReader
	pf         *PositionFile
	pe         *PositionEntry
	m          sync.Mutex
	rotating   bool
	watcher    *fsnotify.Watcher
}

func (i *TailInput) Init(f plugin.ConfigFeeder) (err error) {
	if err = f(&i.conf); err != nil {
		return
	}
	if i.parser, err = parser.Get(i.conf.Format); err != nil {
		return
	}
	if i.conf.TimeFormat != "" {
		i.timeParser, err = parser.NewTimeParser(i.conf.TimeFormat, i.conf.TimeZone)
		if err != nil {
			return
		}
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

func (i *TailInput) Start() (err error) {
	if i.watcher, err = fsnotify.NewWatcher(); err != nil {
		return
	}

	i.open()
	if err = i.Scan(); err != nil {
		return
	}
	go i.eventLoop()
	return
}

func (i *TailInput) eventLoop() {
	tick := time.Tick(10 * time.Second)
	for {
		select {
		case ev := <-i.watcher.Events:
			plugin.Log.Debug(ev)
			if err := i.Scan(); err != nil {
				plugin.Log.Warning(err)
			}
		case err := <-i.watcher.Errors:
			plugin.Log.Warning(err)
		case <-tick:
			i.Scan()
		}
	}
}

func (i *TailInput) open() {
	i.m.Lock()
	defer i.m.Unlock()

	i.rotating = false
	if i.r != nil {
		i.r.Close()
	}

	var err error
	i.pe = i.pf.Get(i.conf.Path)
	i.pe.ReadFromHead = i.conf.ReadFromHead
	if i.r, err = NewPositionReader(i.pe); err != nil {
		plugin.Log.Warning(err, ", wait for creation")
	} else {
		i.watcher.Add(i.conf.Path)
	}
}

func (i *TailInput) Scan() error {
	// To make Scan run only one thread at a time.
	// Also used to block rotation until current scanning completed.
	i.m.Lock()
	defer i.m.Unlock()

	if !i.rotating && i.pe.IsRotated() {
		plugin.Log.Infof("Rotation detected: %s", i.pe.Path)
		var wait time.Duration
		if i.r != nil {
			wait = 5 * time.Second
		}
		i.rotating = true
		time.AfterFunc(wait, i.open)
	}

	if i.r == nil {
		return nil
	}

	for {
		line, _, err := i.r.ReadLine()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		v, err := i.parser.Parse(string(line))
		if err != nil {
			continue
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
	return nil
}

func main() {
	plugin.New(func() plugin.Plugin {
		return &TailInput{}
	}).Run()
}
