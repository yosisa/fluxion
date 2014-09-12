# Fluxion
Fluxion is a streaming data processor written in Go. It's heavily inspired by [Fluentd](http://www.fluentd.org/).

## Why?
We decided to develop fluxion to achieve following goals:

* Fast!
* Small memory footprint
* Easy to use
* Easy to develop

## Build
```bash
make
```

## Run
```bash
export PATH=`pwd`/bundles:$PATH
fluxion
```
