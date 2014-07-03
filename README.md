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

You can install Capstan either by downloading pre-built binaries or building it
from sources.

### Prerequisites: local

You need to have a hypervisor such as QEMU/KVM or VirtualBox installed on your
machine to run local OSv VMs.

If you want to build your own OSv images, you need QEMU installed.

On Fedora:

```
$ sudo yum install qemu-system-x86 qemu-img
```

On OS X:

```
$ brew install qemu
```

### Prerequisites: Google Compute Engine

To run your OSv images on Google Compute Engine, you will need the `gcutil` utility, which is part of the Google Cloud SDK.  Installation instructions are on the [gcutil home page](https://developers.google.com/compute/docs/gcutil/).

### Installing Binaries

To install the binaries, make sure ``$HOME/bin`` is part of the ``PATH``
environment variable and then download the  ``capstan`` executable and place it
in ``$HOME/bin``.

```
$ curl https://raw.githubusercontent.com/cloudius-systems/capstan/master/scripts/download | bash
```

### Installing from Sources

You need a working Go environment installed. See [Go install
instructions](http://golang.org/doc/install.html) for how to do that. Go
version 1.1 or later is required.

Make sure you have the ``GOPATH`` environment variable set to point to a
writable Go workspace such as ``$HOME/go``.

To install Capstan, type:

```
$ go get github.com/cloudius-systems/capstan
```

This installs a ``capstan`` executable to your Go workspace so make sure your
``PATH`` environment variable includes ``$GOPATH/bin``.

For more detailed information, check out [installation instructions](https://github.com/cloudius-systems/capstan/wiki/Capstan-Installation)
on the wiki.

### Updating from Sources

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

To port-forwarding OSv port 22 to Host port 10022, type:

```
$ capstan run -f "10022:22" cloudius/osv
```

To bridging OSv vNIC to Host bridge interface, type:

```
On Fedora:
$ capstan run -n bridge cloudius/osv

On Debian:
$ capstan run -b <manually defined bridge name> cloudius/osv

On OS X with VirtualBox:
$ capstan run -n bridge -b <physical NIC name> cloudius/osv
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
