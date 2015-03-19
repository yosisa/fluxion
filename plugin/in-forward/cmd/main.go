package main

import (
	"github.com/yosisa/fluxion/plugin"
	"github.com/yosisa/fluxion/plugin/in-forward"
)

func main() {
	plugin.New("in-forward", in_forward.Factory).Run()
}
