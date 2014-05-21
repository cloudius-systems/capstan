# Capstanfile

The ``Capstanfile`` is a YAML config file for specifying Capstan images.

An minimal file looks as follows:

```
base: osv-base

cmdline: /tools/hello.so

build: make

files:
  /tools/hello.so: hello.so
```

``base`` specifies the base image that is amended with ``files``.

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

``rootfs`` specifies a directory that is amended to the base image. The direcotry hierarchy
in the base image will be the same as in the rootfs direcotry. If rootfs is not specified
in Capstanfile, a default rootfs direcotry named ROOTFS will be used. If both files and rootfs
are specified, both will be used to amended the base image.
