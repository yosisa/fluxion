SOURCES := $(shell find . -name '*.go')
PKG_SOURCES := $(shell find . -name '*.go' ! -path '*/plugin/*/*' -mindepth 2)
PLUGINS := $(shell find plugin -type f -name main.go | cut -d/ -f2)

all: deps bundles/fluxion plugins

monolithic:
	go build -o bundles/fluxion -tags=monolithic

plugins: $(addprefix bundles/fluxion-,$(PLUGINS))

bundles/fluxion: $(SOURCES)
	go build -o $@

bundles/fluxion-%: plugin/%/*.go plugin/%/*/*.go $(PKG_SOURCES)
	cd plugin/$*/cmd; go build -o ../../../$@

deps:
	go get -t ./...

test: deps
	go test ./...

clean:
	-rm -r bundles

.PHONY: all monolithic plugins deps test clean
