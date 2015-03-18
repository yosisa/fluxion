# Fluxion
[![wercker status](https://app.wercker.com/status/faf5617e6dcfed33610f683d72f9a2fc/s/master "wercker status")](https://app.wercker.com/project/bykey/faf5617e6dcfed33610f683d72f9a2fc)

Fluxion is a streaming data processor written in Go. It's heavily inspired by [Fluentd](http://www.fluentd.org/).

## Why?
We decided to develop fluxion to achieve following goals:

* Fast!
* Small memory footprint
* Easy to use
* Fun to develop

## Build
```bash
make
```

## Run
```bash
export PATH=`pwd`/bundles:$PATH
fluxion
```
