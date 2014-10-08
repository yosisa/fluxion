package main

import (
	"io"
	"path/filepath"
	"strings"
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
	env        *plugin.Env
	conf       *Config
	parser     parser.Parser
	timeParser *parser.TimeParser
	pf         *PositionFile
	watchers   map[string]*Watcher
}

func (i *TailInput) Name() string {
	return "in-tail"
}

func (i *TailInput) Init(env *plugin.Env) (err error) {
	i.env = env
	i.conf = &Config{}
	i.watchers = make(map[string]*Watcher)
	if err = env.ReadConfig(i.conf); err != nil {
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
	go i.pathWatcher()
	return nil
}

func (i *TailInput) pathWatcher() {
	tick := time.Tick(1 * time.Minute)
	for {
		files, err := filepath.Glob(i.conf.Path)
		if err != nil {
			i.env.Log.Error(err)
			return
		}

		changes := make(map[string]bool)
		for f := range i.watchers {
			changes[f] = false
		}
		for _, f := range files {
			if _, ok := changes[f]; ok {
				delete(changes, f)
			} else {
				changes[f] = true
			}
		}

		for f, added := range changes {
			if added {
				i.env.Log.Info("Start watching file: ", f)
				pe := i.pf.Get(f)
				pe.ReadFromHead = i.conf.ReadFromHead
				lp := &LineParser{
					env:        i.env,
					tag:        realTag(i.conf.Tag, pe.Path),
					parser:     i.parser,
					timeParser: i.timeParser,
					timeKey:    i.conf.TimeKey,
				}
				i.watchers[f] = NewWatcher(pe, i.env, lp.parseLine)
			} else {
				i.env.Log.Info("Stop watching file: ", f)
				i.watchers[f].Close()
				delete(i.watchers, f)
			}
		}

		<-tick
	}
}

func realTag(tag, path string) string {
	if !strings.Contains(tag, "*") {
		return tag
	}
	path = strings.Trim(path, "/")
	return strings.Replace(tag, "*", strings.Replace(path, "/", ".", -1), -1)
}

type LineParser struct {
	env        *plugin.Env
	tag        string
	parser     parser.Parser
	timeParser *parser.TimeParser
	timeKey    string
}

func (l *LineParser) parseLine(line []byte) {
	v, err := l.parser.Parse(string(line))
	if err != nil {
		return
	}

	var record *event.Record
	if l.timeKey != "" && l.timeParser != nil {
		if s, ok := v[l.timeKey].(string); ok {
			t, err := l.timeParser.Parse(s)
			if err == nil {
				delete(v, l.timeKey)
				record = event.NewRecordWithTime(l.tag, t, v)
			}
		}
	}
	if record == nil {
		record = event.NewRecord(l.tag, v)
	}
	l.env.Emit(record)
}

type TailHandler func([]byte)

type Watcher struct {
	pe       *PositionEntry
	fsw      *fsnotify.Watcher
	r        *PositionReader
	handler  TailHandler
	rotating bool
	m        sync.Mutex
	notifyC  chan bool
	env      *plugin.Env
}

func NewWatcher(pe *PositionEntry, env *plugin.Env, h TailHandler) *Watcher {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	w := &Watcher{
		pe:      pe,
		fsw:     fsw,
		handler: h,
		notifyC: make(chan bool, 1),
		env:     env,
	}
	w.open()
	go w.eventLoop()
	return w
}

func (w *Watcher) Close() {
	w.fsw.Close()
	close(w.notifyC)
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
		w.env.Log.Warning(err, ", wait for creation")
		w.fsw.Remove(w.pe.Path)
	} else {
		w.r = r
		w.fsw.Add(w.pe.Path)
	}
	w.notify()
}

func (w *Watcher) eventLoop() {
	tick := time.Tick(10 * time.Second)
	for {
		select {
		case _, ok := <-w.notifyC:
			if !ok {
				return
			}
		case ev := <-w.fsw.Events:
			if ev.Op&fsnotify.Create == 0 && ev.Op&fsnotify.Write == 0 {
				continue
			}
		case err := <-w.fsw.Errors:
			w.env.Log.Warning(err)
			continue
		case <-tick:
		}

		if err := w.Scan(); err != nil {
			w.env.Log.Warning(err)
		}
	}
}

func (w *Watcher) Scan() error {
	// To make Scan run only one thread at a time.
	// Also used to block rotation until current scanning completed.
	w.m.Lock()
	defer w.m.Unlock()

	if !w.rotating {
		rotated, truncated := w.pe.IsRotated()
		if rotated {
			w.env.Log.Infof("Rotation detected: %s", w.pe.Path)
			var wait time.Duration
			if w.r != nil {
				wait = 5 * time.Second
			}
			w.rotating = true
			time.AfterFunc(wait, w.open)
		} else if truncated {
			w.env.Log.Infof("Truncation detected: %s", w.pe.Path)
			w.rotating = true
			go w.open()
			return nil
		}
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

func (w *Watcher) notify() {
	select {
	case w.notifyC <- true:
	default:
	}
}

func main() {
	plugin.New(func() plugin.Plugin {
		return &TailInput{}
	}).Run()
}
