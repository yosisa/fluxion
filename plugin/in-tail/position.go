package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"syscall"
)

type PositionEntry struct {
	Offset       int64
	Path         string
	Pos          int64
	Ino          uint64
	ReadFromHead bool
	pf           *PositionFile
}

func (p *PositionEntry) Refresh() int64 {
	fi, err := os.Stat(p.Path)
	if err != nil {
		return 0
	}
	stat := fi.Sys().(*syscall.Stat_t)

	if stat.Ino == p.Ino {
		// The file previously handled by in-tail.
		return p.Pos
	} else if p.Ino != 0 {
		// The file was rotated, safe to read from head of new file.
		p.Set(0, stat.Ino)
		return 0
	}

	// The file handled first time
	var pos int64
	if !p.ReadFromHead {
		pos = fi.Size()
	}
	p.Set(pos, stat.Ino)
	return pos
}

func (p *PositionEntry) IsRotated() bool {
	fi, err := os.Stat(p.Path)
	if err != nil {
		return false
	}
	stat := fi.Sys().(*syscall.Stat_t)
	return stat.Ino != p.Ino
}

func (p *PositionEntry) Set(pos int64, ino uint64) {
	p.Pos = pos
	p.Ino = ino
	p.pf.set(p.Offset, pos, ino)
}

func (p *PositionEntry) SetPos(pos int64) {
	p.Pos = pos
	p.pf.setPos(p.Offset, pos)
}

type PositionFile struct {
	f       *os.File
	entries map[string]*PositionEntry
	m       sync.Mutex
}

func NewPositionFile(path string) (*PositionFile, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	p := &PositionFile{
		f:       f,
		entries: make(map[string]*PositionEntry),
	}
	if err := p.parse(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PositionFile) parse() (err error) {
	var offset int64
	scanner := bufio.NewScanner(p.f)
	for scanner.Scan() {
		line := scanner.Bytes()
		items := bytes.Split(line, []byte{'\t'})
		if len(items) != 3 {
			return errors.New("Invalid position file")
		}

		path := string(items[0])
		entry := &PositionEntry{
			Offset: offset + int64(len(items[0])+1),
			Path:   path,
		}
		entry.Pos, err = strconv.ParseInt(string(items[1]), 16, 64)
		if err != nil {
			return
		}
		entry.Ino, err = strconv.ParseUint(string(items[2]), 16, 64)
		if err != nil {
			return
		}

		p.entries[path] = entry
		offset += int64(len(line) + 1)
	}
	return scanner.Err()
}

func (p *PositionFile) set(offset int64, pos int64, ino uint64) {
	p.m.Lock()
	defer p.m.Unlock()

	p.f.Seek(offset, os.SEEK_SET)
	fmt.Fprintf(p.f, "%016x\t%016x", pos, ino)
}

func (p *PositionFile) setPos(offset int64, pos int64) {
	p.m.Lock()
	defer p.m.Unlock()

	p.f.Seek(offset, os.SEEK_SET)
	fmt.Fprintf(p.f, "%016x", pos)
}

func (p *PositionFile) Get(path string) *PositionEntry {
	p.m.Lock()
	defer p.m.Unlock()
	pe, ok := p.entries[path]
	if !ok {
		pe = p.newEntry(path)
	}
	pe.pf = p
	return pe
}

// Create creates new position entry. This function assumes called inside locked block.
func (p *PositionFile) newEntry(path string) *PositionEntry {
	offset, _ := p.f.Seek(0, os.SEEK_END)
	n, _ := p.f.Write([]byte(path + "\t"))
	offset += int64(n)
	fmt.Fprintf(p.f, "%016x\t%016x\n", 0, 0)
	pe := &PositionEntry{
		Offset: offset,
		Path:   path,
	}
	p.entries[path] = pe
	return pe
}

func (p *PositionFile) Close() error {
	p.m.Lock()
	defer p.m.Unlock()
	return p.f.Close()
}
