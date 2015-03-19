package main

import (
	"github.com/yosisa/fluxion/plugin"
	"github.com/yosisa/fluxion/plugin/in-tail"
)

func main() {
	plugin.New("in-tail", in_tail.Factory).Run()
}
