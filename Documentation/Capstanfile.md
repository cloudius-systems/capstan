# Capstanfile

The ``Capstanfile`` is a YAML config file for specifying Capstan images.

An minimal file looks as follows:

```
base: cloudius/osv-base

cmdline: /tools/hello.so

build: make

files:
  /tools/hello.so: hello.so
```

``base`` specifies the base image that is amended with ``files`` use ``capstan images`` (local images) for a list of available images.

``cmdline`` is the startup command line passed to OSv.

``build`` specifies an optional build command.

``files`` is a map of files that are amended to the base image.  The left side
specifies the full path of the file as it will appear in the image and the
right side specifies a relative path to current directory of the actual file
that is added to the image.

File mapping supports basename variable substitution with the "&" character.
You can specify this:

```
files:
  /usr/app/app.jar: build/&
```

which expands to the following during build:

```
files:
  /usr/app/app.jar: build/app.jar
```

Here ``/usr/app/app.jar`` is the path in the built image and ``build/app.jar``
is the file that is picked up from the local filesystem.

``rootfs`` specifies a directory that is amended to the base image. The directory hierarchy
in the base image will be the same as in the rootfs directory. If rootfs is not specified
in Capstanfile, a default rootfs directory named ROOTFS will be used. If both files and rootfs
are specified, both will be used to amended the base image.

## How to create base images
The easiest way to create Capstan base images is to use the OSv shell script [`build-capstan-base-image`](https://github.com/cloudius-systems/osv/blob/master/scripts/build-capstan-base-image).
The aforementioned script accepts list of OSv apps, creates an image and optionally copies
it to the local Capstan repo like so:
```bash
./scripts/build-capstan-base-image cloudius/python3 python3x 'OSv base with Python 3'
``` 

## RPM support

Capstan also supports installing RPMs with the ``rpm-base`` option:

```
rpm-base:
  name: java-1.7.0-openjdk
  version: 1.7.0.60
  release: 2.4.5.0.fc19
  arch: x86_64
```
