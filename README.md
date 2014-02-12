# Capstan

[![Build Status](https://secure.travis-ci.org/cloudius-systems/capstan.png?branch=master)](http://travis-ci.org/cloudius-systems/capstan)

Capstan is a tool for packing, shipping, and running applications in VMs - just
like Docker but on top of a hypervisor!

## Prerequisite
Install Golang http://golang.org/doc/install#install

## Installation

```
$ cd $GOPATH/src
```

Clone capstan under $GOPATH/src, and then:
```
$ cd capstan
$ go get -v && ./install
```

This will install Capstan to ``$GOPATH/bin`` of your machine.

## Documentation

* [Capstanfile](Documentation/Capstanfile.md)

## Usage

First, you need to push a VM image to your local Capstan repository:

```
$ capstan push <image>
```

You can then launch the image in a VM with:

```
$ capstan run <image>
```

To print a list of images in your repository, do:

```
$ capstan images
```

## License

Capstan is distributed under the 3-clause BSD license.
