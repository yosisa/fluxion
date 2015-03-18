SOURCES := $(shell find . -name '*.go')
PKG_SOURCES := $(shell find . -name '*.go' ! -path '*/plugin/*/*' -mindepth 2)
PLUGINS := $(shell find plugin -type f -name main.go -mindepth 2 -maxdepth 2 | cut -d/ -f2)

all: deps bundles/fluxion plugins

plugins: $(addprefix bundles/fluxion-,$(PLUGINS))

bundles/fluxion: $(SOURCES)
	go build -o $@

bundles/fluxion-%: plugin/%/*.go $(PKG_SOURCES)
	cd plugin/$*; go build -o ../../$@

deps:
	go get -t ./...

test: deps
	go test ./...

clean:
	-rm -r bundles

.PHONY: all plugins deps test clean
