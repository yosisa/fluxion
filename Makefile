SOURCES := $(shell find . -name '*.go')
PKG_SOURCES := $(shell find . -name '*.go' ! -path '*/plugin/*/*' -mindepth 2)
PLUGINS := $(notdir $(shell find plugin -type d -mindepth 1 -maxdepth 1))

all: bundles/fluxion plugins

plugins: $(addprefix bundles/fluxion-,$(PLUGINS))

bundles/fluxion: $(SOURCES)
	go build -o $@

bundles/fluxion-%: plugin/%/*.go $(PKG_SOURCES)
	cd plugin/$*; go build -o ../../$@

clean:
	rm -r bundles

.PHONY: all plugins clean
