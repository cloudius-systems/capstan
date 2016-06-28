# Capstan

Capstan is a tool for rapidly building and running your application on OSv.
Capstan is as simple and fast as using Docker for creating containers, but the
result is a complete lightweight virtual machine image that will run on any
hypervisor with OSv support.

## Features

* Run multiple VMs of an image as copy-on-write
* Linux, OS X, and Windows support
* Hypervisors:
    * QEMU/KVM
    * VirtualBox
    * VMware Workstation and Fusion
* Cloud providers:
    * Google Compute Engine
* Application package management and modular composition of VMs

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

On Ubuntu

```
$ sudo apt-get install qemu-system-x86 qemu-utils
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
$ curl https://raw.githubusercontent.com/cloudius-systems/capstan/master/scripts/download | bash
```

### Installing from Sources

You need a working Go environment installed. See [Go install
instructions](http://golang.org/doc/install.html) for how to do that. Go
version 1.6 or later is required.

Make sure you have the ``GOPATH`` environment variable set to point to a
writable Go workspace such as ``$HOME/go``.

First install godep dependency manager:

```
$ go get github.com/tools/godep
```

This installs a ``godep`` executable to your Go workspace so make sure your
``PATH`` environment variable includes ``$GOPATH/bin``.

This version of Capstan is a form from [original
repository](https://github.com/cloudius-systems/capstan). Because it uses the
same package structure, the easiest way to use the source is to first get the
original version:

```
$ go get github.com/cloudius-systems/capstan
```

Now you can navigate to ``$GOPATH/src/github.com/cloudius-systems/capstan``
and pull from MIKELANGELO repository:

```
$ cd $GOPATH/src/github.com/cloudius-systems/capstan

# Change the URL of the origin.
$ git remote set-url origin https://github.com/mikelangelo-project/capstan.git

# Get the latest release from the new repository.
$ git pull
```

In order to install all dependencies, type:

```
cd $GOPATH/src/github.com/cloudius-systems/capstan
godep restore
```

Your environment is now set. To finally install Capstan, type:

```
cd $GOPATH/src/github.com/cloudius-systems/capstan
go build
```

To install it into your ``GOPATH/bin`` folder, use either ``go install`` or
attached ``./install`` script.

### Updating from Sources

To update capstan to the latest version execute the following commands:
```sh
$ cd $GOPATH/src/github.com/cloudius-systems/capstan
$ git pull
$ go install
```

## Documentation

* [Basic usage](Documentation/Usage.md)
* [Capstanfile](Documentation/Capstanfile.md)
* [Application management](Documentation/ApplicationManagement.md)

## License

Capstan is distributed under the 3-clause BSD license.

## Acknowledgements

This project  has been conducted within the RIA [MIKELANGELO
project](https://www.mikelangelo-project.eu) (no.  645402), started in January
2015, and co-funded by the European Commission under the H2020-ICT- 07-2014:
Advanced Cloud Infrastructures and Services programme.
