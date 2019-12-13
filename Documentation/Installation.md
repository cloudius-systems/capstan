# Step-by-step Installation Guide

You can install Capstan either by downloading pre-built binaries or building it
from sources. You need to install QEMU in both cases.


## Install Prerequisites

### QEMU
Capstan needs QEMU hypervisor being installed on your system, even if you don't intend to
run unikernels with QEMU provider. So go ahead and install it:

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

On FreeBSD:

```
$ sudo pkg install qemu
```

## Install Capstan
Install capstan by manually downloading the relevant binary from
[the latest Github release assets page](https://github.com/cloudius-systems/capstan/releases/latest)
into `$HOME/bin` directory:

You can then use Capstan tool with `$HOME/bin/capstan --help` or include `$HOME/bin` into `PATH` and
use it simply with `capstan --help`.

There you go, happy unikernel creating!

## Install Capstan from source (advanced)

### Install Go 1.11+
Capstan is a Go project and needs to be compiled first. But heads up, compiling Go project is trivial,
as long as you have Go installed. Consult [official documentation](https://golang.org/doc/install)
to learn how to install Go, or use this bash snippet to do it for you:
```bash
curl https://dl.google.com/go/go1.13.5.linux-amd64.tar.gz | sudo tar xz -C /usr/local
sudo mv /usr/local/go /usr/local/go1.13
sudo ln -s /usr/local/go1.13 /usr/local/go

export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$PATH
export PATH=/usr/local/go/bin:$PATH
```

### Compile Capstan
Since Capstan is hosted on GitHub, the compilation process is as simple as:
```
git clone git@github.com:cloudius-systems/capstan.git
cd capstan
export GO111MODULE=on        # Needed for Go 1.11, 1.12
go install
```
That's it, we have Capstan installed. You should be able to use Capstan immediately because it was installed in `$GOPATH/bin` added to your `$PATH` above. To test that it works, try:
```
capstan --help
```

## Maintain Capstan
Capstan is managed with Go Modules as [recommended](https://blog.golang.org/using-go-modules), while developing on Capstan, you should follow these steps:
1. Update import path and corresponding code as you might do
2. Test you changes with `go test ./...`
3. Clear up `go.mod` and `go.sum` file with `go mod tidy`

Then you are ready to tag and release new version. To learn more details about maintaining Go project on Module Mode, see [Go Modules Serial Blogs](https://blog.golang.org/using-go-modules)

## Configure Capstan (advanced)
Capstan uses optimized default values under the hood. But you are allowed to override them with
your own values and this section describes how. Actually, there are three ways to override them
(first non-empty value is taken), although not every variable can be set using all three ways:

### 1) using command-line arguments
You can override some variables using command-line arguments. Please note that you need to repeat
the argument for every command you use, Capstan does not memorize it. Also please pay attention to
the location of the argument. Capstan command must look like this:
```bash
$ capstan {command-line-configuration} other sub commands and args
# For example:
$ capstan -r v0.53.0 package compose img1 --size 10GB
          |- here --|
```

List of supported arguments:

* `-r <release-tag>` specifies the OSv github release tag that is used to fetch precompiled
kernel and packages from.

### 2) using configuration file
Capstan supports configuration file to permanently override some internal defaults. This file is
located in `$HOME/.capstan/config.yaml`. It is not created by default, so you need to create the file
and folder if they do not exist yet. The file is nothing but a simple yaml containing "key: value"
pairs e.g.
```yaml
repo_url: https://mikelangelo-capstan.s3.amazonaws.com/
disable_kvm: false
qemu_aio_type: threads
```
List of supported keys:

* `repo_url` overrides the default remote S3 repository URL that is used to fetch precompiled
packages from if `--s3` flag is enabled.
* `disable_kvm` by default KVM acceleration is turned on to speed up unikernel creation, but in
certain circumstances this results in error. Set this to `true` if you have problems using KVM.
* `qemu_aio_type` by default QEMU aio type is set to `threads` for compatibility reasons. A faster
aio option may be `native` depending on the version of QEMU, but it's not supported on all platforms.

Please note that if command line argument is used to override the same value (e.g. -u for repository
URL), then the value from configuration file is ignored.

### 3) using environment variables
Capstan reads execution environment to override internal variables. Just set environment variable
prior to calling Capstan commands, for example:
```bash
$ export CAPSTAN_REPO_URL=https://mikelangelo-capstan.s3.amazonaws.com/
$ capstan package compose ...
```

List of supported environment variables:

* `CAPSTAN_REPO_URL` overrides the default remote repository URL that is used to fetch precompiled
packages from.
* `DISABLE_KVM` [true|false]
* `QEMU_AIO_TYPE` [threads|native]

Please note that environment variables have the lowest priority - if same variable is set using either
command-line argument or configuration file, then environment variable is ignored.

### Double-check your configuration
There is a Capstan command to double-check which configuration values are eventually used:
```
capstan config print
```