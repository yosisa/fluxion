SOURCES := $(shell find . -name '*.go')
PLUGINS := $(notdir $(shell find plugin -type d -mindepth 1 -maxdepth 1))

all: bundles/fluxion plugins

plugins: $(addprefix bundles/fluxion-,$(PLUGINS))

bundles/fluxion: $(SOURCES)
	go build -o $@

bundles/fluxion-%: plugin/% $(SOURCES)
	cd $<; go build -o ../../$@

.PHONY: all plugins
