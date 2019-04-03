# Runtime `native`
This document describes how to write a valid `meta/run.yaml` configuration file
for running **native C/C++** application. This is the most basic runtime that depends
on no MPM package, but rather only invokes the corresponding .so.

## Running own C application
Bear in mind that you need to compile your application source code on your own, Capstan
won't do it for you. Furthermore, the system that you're compiling your application on must be compatible
with the system that base unikernel (mike/osv-loader) was built on, notice the "Platform" column:

```bash
$ capstan images
Name             Description       Version   Created             Platform
mike/osv-loader  OSv Bootloader    d9a8771   2017-11-16 07:46    Ubuntu-14.04
                                                                     ^                           
```

This basically means that you need to compile your application on Ubutnu 14.04. If compiling, say, on Ubuntu 16.04,
then application will most probably crash on OSv. Also, you need to use `-fPIC -shared` flags when compiling. Please
navigate [here](https://github.com/mikelangelo-project/osv-utils-xlab) to see some example C applications (echo.c,
sleep.c, cat.c, etc.) as well as corresponding Makefile.

Once application is compiled, you only need to copy the resulting `.so` in Capstan package directory and provide the boot
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