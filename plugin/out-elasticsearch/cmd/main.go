package main

import (
	"github.com/yosisa/fluxion/plugin"
	"github.com/yosisa/fluxion/plugin/out-elasticsearch"
)

func main() {
	plugin.New("out-elasticsearch", out_elasticsearch.Factory).Run()
}
