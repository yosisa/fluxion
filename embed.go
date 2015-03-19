// +build monolithic

package main

import (
	"github.com/yosisa/fluxion/plugin"
	"github.com/yosisa/fluxion/plugin/filter-js"
	"github.com/yosisa/fluxion/plugin/in-forward"
	"github.com/yosisa/fluxion/plugin/in-tail"
	"github.com/yosisa/fluxion/plugin/out-elasticsearch"
	"github.com/yosisa/fluxion/plugin/out-file"
	"github.com/yosisa/fluxion/plugin/out-forward"
	"github.com/yosisa/fluxion/plugin/out-stdout"
)

func init() {
	plugin.EmbeddedPlugins["filter-js"] = filter_js.Factory
	plugin.EmbeddedPlugins["in-forward"] = in_forward.Factory
	plugin.EmbeddedPlugins["in-tail"] = in_tail.Factory
	plugin.EmbeddedPlugins["out-elasticsearch"] = out_elasticsearch.Factory
	plugin.EmbeddedPlugins["out-file"] = out_file.Factory
	plugin.EmbeddedPlugins["out-forward"] = out_forward.Factory
	plugin.EmbeddedPlugins["out-stdout"] = out_stdout.Factory
}
