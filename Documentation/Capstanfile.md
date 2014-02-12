# Capstanfile

The ``Capstanfile`` is a YAML config file for specifying Capstan images.

An minimal file looks as follows:

```
base: osv-base

cmdline: /tools/hello.so

files:
  /tools/hello.so: hello.so
```

``base`` specifies the base image that is amended with ``files``.

``cmdline`` is the startup command line passed to OSv.

``files`` is a map of files that are amended to the base image.  The left side
specifies the full path of the file as it will appear in the image and the
right side specifies a relative path to current directory of the actual file
that is added to the image.
