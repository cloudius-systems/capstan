# Capstan

[![Build Status](https://secure.travis-ci.org/cloudius-systems/capstan.png?branch=master)](http://travis-ci.org/cloudius-systems/capstan)

Capstan is a tool for packing, shipping, and running applications in VMs - just
like Docker but on top of a hypervisor!

## Installation

You need a working Go environment installed. See [Go install
instructions](http://golang.org/doc/install.html) for how to do that.

To install Capstan, type:

```
$ go get github.com/cloudius-systems/capstan/capstan
```

This installs a ``capstan`` executable to your Go workspace so make sure your
``PATH`` environment variable includes ``$GOPATH/bin``.

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

## Documentation

* [Capstanfile](Documentation/Capstanfile.md)

## License

Capstan is distributed under the 3-clause BSD license.
