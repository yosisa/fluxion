package main

import (
	"flag"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/engine"
)

var config struct {
	Buffer []*buffer.Options
	Input  []map[string]interface{}
	Filter []map[string]interface{}
	Output []map[string]interface{}
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "c", "/etc/fluxion.toml", "config file")
	flag.Parse()

	if _, err := toml.DecodeFile(configPath, &config); err != nil {
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
	for _, conf := range config.Output {
		eng.RegisterOutputPlugin(conf)
	}
	eng.Start()
	eng.Wait()
}
