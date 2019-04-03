# Filesystems

OSv unikernel supports multiple filesystems - ZFS (Zeta File System), ROFS (Read-Only File System) and RamFS.
This means that user can build an image, run it and have OSv mount one of these filesystems. 

## ZFS
ZFS is the original and default filesystem OSv images built by Capstan come with. ZFS is very performant read-write file system
ideal for running stateful apps on OSv when data needs to be changed and persisted. For more details please
read [this](https://github.com/cloudius-systems/osv/wiki/ZFS) and [that](https://github.com/cloudius-systems/osv/wiki/Managing-your-storage).

## ROFS
Read-Only Filesystem has been added recently to OSv and requires 
[latest version 0.51](https://github.com/cloudius-systems/osv/releases/tag/v0.51.0). As the name suggests ROFS allows only
reading data from files and therefore well suites running stateless applications on OSv when only code has to be accessed. For more details
look at [this commit](https://github.com/cloudius-systems/osv/commit/cd449667b7f86721095ddf4f9f3f8b87c1c414c9) and
[original Python script](https://github.com/cloudius-systems/osv/blob/master/scripts/gen-rofs-img.py).

## Composing packages
When composing OSv images using ```capstan package compose```, please select desired filesystem using new ```-fs``` option like so:
```
capstan package compose --fs rofs <my-package-name>
```
