package main

import (
	"github.com/yosisa/fluxion/plugin"
	"github.com/yosisa/fluxion/plugin/filter-js"
)

func main() {
	plugin.New("filter-js", filter_js.Factory).Run()
}
