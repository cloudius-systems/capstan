# Application Management

The purpose of this document is to guide you through the first steps of the
application management using a tool developed in
[MIKELANGELO](https://mikelangelo-project.eu). The tool is based on
[Capstan](https://github.com/cloudius-systems/capstan),
provided by ScyllaDB as part of their support for building lightweight
OSv-based Virtual Machines.

*Please note: this preliminary version of the documentation may omit/miss important information. If you have any comments or suggestions about the tool, the packages or this documentation, feel free to let us know.*

## Installation

You must first install Capstan from [this
repository](https://github.com/mikelangelo-project/capstan). You can install
binary version or follow the instructions for build it from source code. Ensure
the ``capstan`` is in your ``$PATH``.

This version of Capstan currently does not support automatic downloading of
required packages. In order to install packages locally, first download them
from [MIKELANGELO project
page](https://mikelangelo-project.eu/mikelangelo-packages.v0.2.0.tar.gz). Then
import them into Capstan's local repository (always at ``$HOME/.capstan``):

```
$ tar -C ~/ -xvzf mikelangelo-packages.v0.2.0.tar.gz
```

This will extract the packages and the required OSv launcher VM, a small VM
built by the OSv build system. It contains only a ramdisk with the kernel and
some additional tools required to compose applications in the following stages.

## Help

Capstan tool provides several commands which are conveniently described in the
built-in help. To get an overview of all commands, add -h switch to the
command. For example:

```
$ capstan -h

NAME:
   capstan - pack, ship, and run applications in light-weight VMs

USAGE:
   capstan [global options] command [command options] [arguments...]

COMMANDS:
   info         show disk image information
   import       import an image to the local repository
   pull         pull an image from a repository
   rmi          delete an image from a repository
   run          launch a VM. You may pass the image name as the first argument.
   build        build an image
   compose      compose the image from a folder or a file
   images, i    list images
   search       search a remote images
   instances, I list instances
   stop         stop an instance
   delete       delete an instance
   package      package manipulation tools
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   -u "https://s3.amazonaws.com/osv.capstan/"   remote repository URL
   --help, -h                                   show help
   --version, -v                                print the version
```

Each of the available commands furthermore provides additional help, for example

```
$ capstan compose -h

NAME:
   capstan compose - compose the image from a folder or a file

USAGE:
   capstan compose [command options] [arguments...]

OPTIONS:
   --loader_image, -l "mike/osv-loader" the base loader image
   --size, -s "10G"                   size of the target user partition (use M or G suffix)
```

## Package management

Package management consists of the following commands:

```
$ capstan package -h

NAME:
   capstan package - package manipulation tools

USAGE:
   capstan package command [command options] [arguments...]

COMMANDS:
   init        initialise package structure
   build       builds the package into a compressed file
   compose     composes the package and all its dependencies into OSv image
   collect     collects contents of this package and all required packages
   list        lists the available packages
   import      builds the package at the given path and imports it into a chosen repository
   help, h     Shows a list of commands or help for one command
```

The following subsections explain these commands in detail.

### Package initialisation

A package is any directory in your file system that contains a special package
manifest file. The file should be formatted as
[YAML](http://www.yaml.org/start.html) and stored in ``meta/package.yaml`` relative
to the package root.

There are two ways to initialise a package:

* Manually edit the file in meta/package.yaml.

* Using the capstan tool to initialise the package.

#### Manual initialisation of a package

Using your favourite text editor, open a file ``meta/package.yaml``:

``
$ mkdir meta
$ vim meta/package.yaml
``

Paste the following content into the file:

``
name: com.example.app
title: Example App
author: Example Author
``

This is the smallest possible package manifest file. It defines an unique name
of the application, the title of this application and the author. The suggested
convention for naming packages and applications is to use the reversed domain
notation as in the example above, however this is not a strict requirement. The
title and the author are not used at this moment, however they will be used for
package directory provided in future versions.

#### Initialisation with Capstan

To initialise a package using Capstan tool, one can simply provide all the
necessary information in a single command, like:

```
$ capstan package init --name "com.example.app" --title "Example App" --author "Example User"
```

This will create a meta subdirectory and ``meta/package.yaml`` file with the
given content.

### Working with dependencies

Capstan package initialisation command allows one to optionally specify one or
more required packages. These package will be included when the application
will be composed into a VM. For example, to include the CLI (Command Line
Interface) tool in our own application, initialise a package using
``--require`` option:

```
$ capstan package init --name "com.example.app" --title "Example App" --author "Example User" --require eu.mikelangelo-project.osv.cli
```

The same can be achieved by manually adding one or more required packages in ``meta/package.yaml``:

```
name: eu.mikelangelo-project.app.demo
title: DEMO App
author: lemmy
require:
    - eu.mikelangelo-project.osv.cli
```

When applications are composed, the required packages are recursively inspected
and the content of all of them is added to the application. For example, the
``eu.mikelangelo-project.osv.cli`` package requires
``eu.mikelangelo-project.osv.httpserver`` which gets added automatically as
well.

A package may override the content of any of its required packages which allows
users to customise or to reconfigure one of the base packages.

### Listing available packages

To list all packages available in your local repository, use ``capstan package
list`` command. For each of the packages, its name, description (title), version
and time of creation will be displayed. Use this command to find out how to
refer to required packages.

```
$ capstan package list

Name                                               Description                                        Version                   Created        
eu.mikelangelo-project.app.hadoop-hdfs             Hadoop HDFS                                        2.7.2                     2016-03-30T10:23:03+02:00
eu.mikelangelo-project.openfoam.core               OpenFOAM Core                                      2.4.0                     2016-02-23T19:24:36+01:00
eu.mikelangelo-project.openfoam.simplefoam         OpenFOAM simpleFoam                                2.4.0                     2016-01-19T09:39:12+01:00
eu.mikelangelo-project.osv.bootstrap               OSv Bootstrap                                      0.24-46-g464f4e0          2016-01-20T05:43:24+01:00
eu.mikelangelo-project.osv.cli                     OSv Command Line Interface                         0.24-46-g464f4e0          2016-01-20T05:43:42+01:00
eu.mikelangelo-project.osv.httpserver              OSv HTTP REST Server                               0.24-46-g464f4e0          2016-01-20T05:43:47+01:00
eu.mikelangelo-project.osv.java                    Java JRE 1.7.0                                     1.7.0-openjdk-1.7.0.60-2.4.7.4.fc20.x86_64 2016-03-22T17:39:09+01:00
```

### Collecting package content

Collecting package content allows you to inspect the content of the application
package exactly as it will be uploaded into target VM without actually
uploading it. The content is collected into a subdirectory named ``mpm-pkg``.
It is not necessary to delete this directory as it is ignored by all package
related commands.

To collect a package using Capstan, simply execute the following command at the root of package:

```
$ capstan package collect
```

### Building a package

Building a package creates a TAR archive of the entire package content,
including its metadata. The archive can be shared with other users who can
simply import it into their own package repository
(``$HOME/.capstan/packages``).

### Importing a package

By importing a package into your local package repository, you will be able to
use it when composing other packages. Simply execute:

```
$ capstan package import 
```

Use ``capstan package list`` to verify the package has been properly imported
into your local package repository.

### Package composition

Package composition takes the content of the package and all of its required
packages and creates a new QCOW2 virtual machine image. Current version of the
command supports the following additional configuration options:

* ``--size, -s``: specify the size of the target VM. Human readable representation can be used, for example 1G or 512M to request a 1 GB or 512 MB image.

* ``--update``: request an update of an existing VM. See below for more details

* ``--run``: specify the default run command to be used when starting a VM. It will be read by the OSv loader and executed immediately after the kernel is booted

* ``--verbose``: get detailed information about the files that are being uploaded onto the VM

To compose a VM image, simply execute

```
$ capstan package compose [image-name]
```

Here ``image-name`` can be arbitrary name of the target image, for example ``hello/example-app``.

### Updating existing virtual machine images

When making small changes to the application content, it is inefficient to
compose a VM from scratch every time. Thus, the ``--update`` command line option
allows one to request composition by uploading only the files that have been
modified since the last run. If the target image (``image-name``) does not
exist, it will be created and all files will be uploaded. However, if it
already exists, a file hash cache will be consulted to determine which files
need to be uploaded.

**IMPORTANT**: current version does not support removal of files or
directories. If such an operation is required, ``--update`` should not be used.
Furthermore, modifications are determined only an SHA1 hash of the files on the
host composing the VM images. If any of the files have been changed on the VM
itself, this will not be detected with this mechanism.

## Running applications

Once we have a full VM stored in our local repository, we can launch it by
using capstan run command. If we have composed an application with name
``hello/example-app``, we can launch it with:

```
$ capstan run hello/example-app
```

This will execute whatever the previous command was set to. In case you have
not specified the ``--run`` command when composing the image, this is not going
to be what you wanted to do, as it will actually format VM’s entire root disk
:-).

Instead, you should specify the run command either during image composition or
runtime:

```
$ capstan package compose --run /usr/bin/myapp hello/example-app
$ capstan run hello/example-app
```

or

```
$ capstan package compose hello/example-app
$ capstan run -e /usr/bin/myapp hello/example-app
```

If you have included CLI into your application, you may launch it right away:

```
$ capstan run -e /cli/cli.so

Updating image ``/home/lemmy/.capstan/repository/eu.mikelangelo-project.app.demo/eu.mikelangelo-project.app.demo.qemu...
Setting cmdline: /tools/cpiod.so --prefix /
Uploading files 287 / 287 [====================================================] 100.00 % 0
All files uploaded
Created instance: eu.mikelangelo-project.app.demo
Setting cmdline: /cli/cli.so
OSv v0.24-78-g69bd35e
eth0: 192.168.122.15
/# exit
Goodbye
```

## Java applications

Capstan provides support for composing and running Java-based applications. To
enable Java application, one must add a dependency to
``eu.mikelangelo-project.osv.java``, for example:

```
name: eu.mikelangelo-project.app.hellojava
title: Hello Java
author: lemmy
require:
    - eu.mikelangelo-project.osv.java
```

Additionally, you have to provide another manifest file configuring the Java
application. This manifest file consists of the following options:

```
* main: fully classified name of the main class
* args: a list of command line args used by the application
* classpath: a list of paths where classes and other resources should be found
* vmargs: a list of JVM args (for example Xmx, Xms, …)
```

This manifest must be stored in ``meta/java.yaml`` file. An example of a simple
Java manifest is:

```
main: main.Hello
classpath:
    - /
```

This will start class ``main.Hello``. Classpath is set to the root because the main
class is located in ``/main/Hello.class`` file.

A slightly more complex example of Java manifest (taken from our Hadoop HDFS
application; note that classpath is trimmed).

```
main: org.apache.hadoop.hdfs.server.datanode.DataNode
classpath: 
    - /hdfs/etc/hadoop
    - /hdfs/share/hadoop/common/lib/commons-logging-1.1.3.jar
    - /hdfs/share/hadoop/common/lib/jersey-json-1.9.jar
    - ...
vmargs: 
    - Dproc_datanode 
    - Xmx1000m 
    - Djava.net.preferIPv4Stack=true 
    - Dhadoop.log.dir=/hdfs/logs 
    - Dhadoop.log.file=hadoop.log 
    - Dhadoop.home.dir=/hdfs 
    - Dhadoop.id.str=xlab 
    - Dhadoop.root.logger=INFO,console 
    - Djava.library.path=/hdfs/lib/native 
    - Dhadoop.policy.file=hadoop-policy.xml 
    - Djava.net.preferIPv4Stack=true 
    - Dhadoop.security.logger=ERROR,RFAS 
    - Dhadoop.security.logger=INFO,NullAppender 
```
