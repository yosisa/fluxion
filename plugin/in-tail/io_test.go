package in_tail

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestPositionReader(t *testing.T) {
	posfileName, posfile := tempfile(t)
	posfile.Close()
	defer os.Remove(posfileName)
	pf, err := NewPositionFile(posfileName)
	if err != nil {
		t.Fatal(err)
	}

	logfileName, logfile := tempfile(t)
	defer os.Remove(logfileName)
	fmt.Fprintf(logfile, "foo\n\n")
	fmt.Fprintf(logfile, "ba")

	pe := pf.Get(logfileName)
	pe.ReadFromHead = true
	r, err := NewPositionReader(pe)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if pe.Pos != 0 {
		t.Fatalf("Invalid position: expected 0 but %d", pe.Pos)
	}

	line, err := r.ReadLine()
	if string(line) != "foo" || err != nil {
		t.Fatalf("Invalid line: expected 'foo' but '%s', err=%v", line, err)
	}
	if pe.Pos != 4 {
		t.Fatalf("Invalid position: expected 4 but %d", pe.Pos)
	}

	line, err = r.ReadLine()
	if line != nil || err != io.EOF {
		t.Fatalf("Invalid line: expected nil but '%s', err=%v", line, err)
	}
	if pe.Pos != 5 {
		t.Fatalf("Invalid position: expected 5 but %d", pe.Pos)
	}

	fmt.Fprintf(logfile, "r")
	line, err = r.ReadLine()
	if line != nil || err != io.EOF {
		t.Fatalf("Invalid line: expected nil but '%s', err=%v", line, err)
	}
	if pe.Pos != 5 {
		t.Fatalf("Invalid position: expected 5 but %d", pe.Pos)
	}

	fmt.Fprintf(logfile, "\n")
	line, err = r.ReadLine()
	if string(line) != "bar" || err != nil {
		t.Fatalf("Invalid line: expected 'bar' but '%s', err=%v", line, err)
	}
	if pe.Pos != 9 {
		t.Fatalf("Invalid position: expected 9 but %d", pe.Pos)
	}

	line, err = r.ReadLine()
	if line != nil || err != io.EOF {
		t.Fatalf("Invalid line: expected nil but '%s', err=%v", line, err)
	}

	// CRLF
	fmt.Fprintf(logfile, "\r\nba")
	line, err = r.ReadLine()
	if line != nil || err != io.EOF {
		t.Fatalf("Invalid line: expected nil but '%s', err=%v", line, err)
	}
	if pe.Pos != 11 {
		t.Fatalf("Invalid position: expected 11 but %d", pe.Pos)
	}

	fmt.Fprintf(logfile, "z\r\nx")
	line, err = r.ReadLine()
	if string(line) != "baz" || err != nil {
		t.Fatalf("Invalid line: expected 'baz' but '%s', err=%v", line, err)
	}
	if pe.Pos != 16 {
		t.Fatalf("Invalid position: expected 16 but %d", pe.Pos)
	}
}

func tempfile(t *testing.T) (string, *os.File) {
	f, err := ioutil.TempFile("", "fluxion-in-tail")
	if err != nil {
		t.Fatal(err)
	}
	return f.Name(), f
}
