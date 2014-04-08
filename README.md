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
    * VMware Workstation and Fusion
* Cloud providers:
    * Google Compute Engine

## Installation

* [Capstan-Installation](https://github.com/cloudius-systems/capstan/wiki/Capstan-Installation)

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
