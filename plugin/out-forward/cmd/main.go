package main

import (
	"github.com/yosisa/fluxion/plugin"
	"github.com/yosisa/fluxion/plugin/out-forward"
)

func main() {
	plugin.New("out-forward", out_forward.Factory).Run()
}
