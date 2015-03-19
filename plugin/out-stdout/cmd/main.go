package main

import (
	"github.com/yosisa/fluxion/plugin"
	"github.com/yosisa/fluxion/plugin/out-stdout"
)

func main() {
	plugin.New("out-stdout", out_stdout.Factory).Run()
}
