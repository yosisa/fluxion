package out_file

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"text/template"

	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/message"
	"github.com/yosisa/fluxion/plugin"
)

type Config struct {
	Path   string
	Format string
}

type OutFile struct {
	env  *plugin.Env
	conf Config
	tmpl *template.Template
	ino  uint64
	w    *os.File
}

func (p *OutFile) Init(env *plugin.Env) (err error) {
	p.env = env
	if err = env.ReadConfig(&p.conf); err != nil {
		return
	}
	if p.conf.Format != "" {
		p.tmpl, err = template.New("").Parse(p.conf.Format)
	}
	return
}

func (p *OutFile) Start() error {
	return nil
}

func (p *OutFile) Encode(ev *message.Event) (buffer.Sizer, error) {
	var err error
	b := new(bytes.Buffer)
	if p.tmpl != nil {
		err = p.tmpl.Execute(b, ev)
		b.Write([]byte{'\n'})
	} else {
		fmt.Fprintf(b, "[%s] %v: ", ev.Tag, ev.Time)
		err = json.NewEncoder(b).Encode(ev.Record)
	}
	if err != nil {
		return nil, err
	}
	return buffer.BytesItem(b.Bytes()), nil
}

func (p *OutFile) Write(l []buffer.Sizer) (int, error) {
	if err := p.reopen(); err != nil {
		return 0, err
	}
	for i, b := range l {
		if _, err := p.w.Write(b.(buffer.BytesItem)); err != nil {
			return i, err
		}
	}
	return len(l), nil
}

func (p *OutFile) reopen() error {
	ino, err := inode(p.conf.Path)
	if err != nil {
		return err
	}
	if p.ino != ino && p.w != nil {
		p.env.Log.Infof("Rotation detected: %s", p.conf.Path)
		p.w.Close()
		p.w = nil
	}
	if p.w == nil {
		w, err := os.OpenFile(p.conf.Path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		p.w = w
		if ino == 0 {
			if ino, err = inode(p.conf.Path); err != nil {
				return err
			}
		}
		p.ino = ino
	}
	return nil
}

func (p *OutFile) Close() error {
	return p.w.Close()
}

func inode(path string) (ino uint64, err error) {
	var fi os.FileInfo
	fi, err = os.Stat(path)
	if err == nil {
		ino = fi.Sys().(*syscall.Stat_t).Ino
	} else if os.IsNotExist(err) {
		err = nil
	}
	return
}

func Factory() plugin.Plugin {
	return &OutFile{}
}
