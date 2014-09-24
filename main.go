package main

import (
	"flag"
	"io/ioutil"
	"log"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/engine"
)

var config struct {
	Buffer []*buffer.Options
	Input  []map[string]interface{}
	Filter []map[string]interface{}
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "c", "/etc/fluxion.toml", "config file")
	flag.Parse()

	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}
	_, err = toml.Decode(string(b), &config)
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}

	eng := engine.New()
	for _, bopts := range config.Buffer {
		eng.RegisterBuffer(bopts)
	}
	for _, conf := range config.Input {
		eng.RegisterInputPlugin(conf)
	}
	for _, conf := range config.Filter {
		eng.RegisterFilterPlugin(conf)
	}

	// To support `output:...` form, re-decoding with relax type is needed.
	var c map[string][]map[string]interface{}
	toml.Decode(string(b), &c)
	for k, v := range c {
		keys := strings.SplitN(k, ":", 2)
		if strings.ToLower(keys[0]) != "output" {
			continue
		}

		var name string
		if len(keys) == 2 {
			name = keys[1]
		}
		for _, conf := range v {
			eng.RegisterOutputPlugin(name, conf)
		}
	}

	eng.Start()
	eng.Wait()
}
