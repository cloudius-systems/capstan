# Capstan

[![Build Status](https://secure.travis-ci.org/cloudius-systems/capstan.png?branch=master)](http://travis-ci.org/cloudius-systems/capstan)

Capstan is a tool for rapidly building and running your application on OSv.
Capstan is as simple and fast as using Docker for creating containers, but the
result is a complete virtual machine image that will run on any hypervisor with
OSv support.

## Features

* Run multiple VMs of an image as copy-on-write
* Linux, OS X, and Windows support
* Hypervisors:
    * QEMU/KVM
    * VirtualBox
* Cloud providers:
    * Google Compute Engine

## Installation

You need a working Go environment installed. See [Go install
instructions](http://golang.org/doc/install.html) for how to do that. Go
version 1.1 or later is required.

You also need QEMU installed. On Fedora:

```
$ sudo yum install qemu-system-x86
```

On OS X:

```
$ brew install qemu
```

To install Capstan, type:

```
$ go get github.com/cloudius-systems/capstan/capstan
```

This installs a ``capstan`` executable to your Go workspace so make sure your
``PATH`` environment variable includes ``$GOPATH/bin``.

## Updating

To update capstan to the latest version execute the following commands:
```sh
$ cd $GOPATH/src/github.com/cloudius-systems/capstan
$ git pull
$ ./install
```

## Usage

To run OSv on default hypervisor which is QEMU/KVM, type:

```
$ capstan run cloudius/osv
```

To run OSv on VirtualBox, type:

```
$ capstan run -p vbox cloudius/osv
```

To show a list of available remote images, type:

```
$ capstan search
```

To show a list of locally installed images, type:

```
$ capstan images
```

## Documentation

* [Capstanfile](Documentation/Capstanfile.md)

## Examples

* [Running native Linux apps on OSv](https://github.com/cloudius-systems/capstan-example)
* [Running Java on OSv](https://github.com/penberg/capstan-example-java)
* [Running Clojure on OSv](https://github.com/tzach/capstan-example-clojure)

## License

Capstan is distributed under the 3-clause BSD license.
