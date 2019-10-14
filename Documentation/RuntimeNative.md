# Runtime `native`
This document describes how to write a valid `meta/run.yaml` configuration file
for running **native Linux** application. This is the most basic runtime that does not depend
on any intermediate runtime MPM package (Java, Python, etc), but rather launches the corresponding
native ELF x86_64 file directly.

## Running Linux x86_64 applications
Bear in mind that you need to either use the application binaries from the Linux host or compile it from 
the source code on your own, Capstan won't do it for you.

Using pre-built binaries from Linux host is the easiest way - you just need to identify all the binaries
and any libraries it depends on and relevant configuration files and copy all those to your capstan
project directory. Also make sure that the Linux binaries are standard Linux position-independent executables ("PIE")
or position-dependent executables (non-relocatable dynamically linked executable) or shared libraries - 
(for details please see [OSv Linux compatibility doc](https://github.com/cloudius-systems/osv/wiki/OSv-Linux-ABI-Compatibility)).
One way to automate this process is to use [this OSv shell script](https://github.com/cloudius-systems/osv/blob/master/scripts/manifest_from_host.sh) like
so:
```bash
$ ./scripts/manifest_from_host.sh -w ls
$ ./scripts/build -j4 --append-manifest export=selected usrskel=none
```

The resulting files from `build/export` directory have to be copied to the capstan project directory.
Once all application binaries and configuration files are placed in the capstan project directory,
you need to create `meta/run.yaml` to specify the application boot command: 

```yaml
# meta/run.yaml
runtime: native
config_set:
  default:
    bootcmd: "/ls -l"
config_set_default: default
```

and compose the unikernel image like so:
```bash
$ capstan package compose demo
$ capstan run demo
```

If you decide to compile it, for the best results the system that you're compiling your application on 
should be compatible with the system the base unikernel (mike/osv-loader) was built on, notice the "Platform" column:

```bash
$ capstan images
Name             Description       Version   Created             Platform
mike/osv-loader  OSv Bootloader    d9a8771   2017-11-16 07:46    Ubuntu-14.04
                                                                     ^                           
```

This basically means that you should compile your application on Ubuntu 14.04 based on this example. If compiling, say, on Ubuntu 16.04,
then application may crash on OSv. Also, you need to use `-fPIC -shared` or `-fpie -pie` flags when compiling to create a
shared library or standard Linux position-independent executables ("PIE") and position-dependent executables (non-relocatable dynamically linked executable) - 
(for details please see [OSv Linux compatibility doc](https://github.com/cloudius-systems/osv/wiki/OSv-Linux-ABI-Compatibility)). Please
navigate [here](https://github.com/mikelangelo-project/osv-utils-xlab) to see some example C applications (echo.c,
sleep.c, cat.c, etc.) as well as corresponding Makefile.

Once application is compiled, you only need to copy the resulting `.so` in Capstan project directory and provide the boot
command to run it. Following configuration can be used to run your .so application inside OSv, `echo.so` in this example:

```yaml
# meta/run.yaml

runtime: native

config_set:
  say_hey:
    bootcmd: /echo.so Hey hey!
```

Example:

```bash
$ capstan package compose demo
$ capstan run demo --boot say_hey
Command line will be set based on --boot parameter
Created instance: demo
Setting cmdline: runscript /run/echo
OSv v0.24-434-gf4d1dfb
eth0: 192.168.122.15
Hey hey!
```