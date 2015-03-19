package main

import (
	"github.com/yosisa/fluxion/plugin"
	"github.com/yosisa/fluxion/plugin/out-file"
)

func main() {
	plugin.New("out-file", out_file.Factory).Run()
}
