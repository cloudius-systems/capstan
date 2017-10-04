# Capstan

*This is an upgraded version of the [original Capstan](https://github.com/cloudius-systems/capstan) and
is maintained by [MIKELANGELO consortium](https://www.mikelangelo-project.eu). Although it is still
possible, joining this repository with the original Capstan is not very likely. We're looking towards
the first stable release at the moment.*

Capstan is a command-line tool for rapidly running your application on [OSv unikernel](http://osv.io).
It focuses on improving user experience during building the unikernel and attempts to support
not only a variety of runtimes (C, C++, Java, NodeJS etc.), but also a variety of ready-to-run
applications (Hadoop HDFS, MySQL, SimpleFOAM etc.).

## Philosophy
Buildning unikernels is generally a nightmare! It is a non-trivial task that requires deep
knowledge of unikernel implementation. It depends on numerous installation tools and takes
somewhat 10 minutes to prepare each unikernel once configured correctly.
But an application-oriented developer is not willing to take a load of new knowledge about unikerel
specifics, nor wait long minutes to compile! And that's where Capstan comes in.

Capstan tends to be a tool that one configures with *application-oriented settings*
(Where is application entry point? What environment variables to pass? etc.) and then
runs a command or two to quickly boot up a new unikernel with application. Measured in seconds.

To achieve this, Capstan uses **precompiled** artefacts: precompiled OSv kernel, precompiled Java runtime,
precompiled MySQL, and many more. All you have to do is to name what precompiled packages you want
to have available in your unikernel and that's it.

## Features
Capstan is designed to prepare and run OSv unikernel for you.
With Capstan it is possible to:

* prepare OSv unikernel without compiling anything but your application, in seconds
* use any precompiled package from the MIKELANGELO package repository, or a combination thereof
* set arbitrary size of the target unikernel filesystem
* run OSv unikernel using one of the supported providers

But Capstan is not a magic tool that could solve all the problems for you.
Capstan does **not**:

* compile your application. If you have Java application, you need to use `javac` compiler and compile
the application yourself prior using Capstan tool!
* inspect your application. Capstan does nothing with your application but copies it into the unikernel
* overcome OSv unikernel limits. Consult OSv documentation about what these limits are since they
all still apply. Most notably, you can only run single process inside unikernel (forks are forbidden).

## Getting started
Capstan can be installed using precompiled binary or compiled from source.
<br>
[Step-by-step Capstan Installation Guide](Documentation/Installation.md)

Using Capstan is rather simple: open up your project directory and create
[Capstan configuration files](Documentation/ConfigurationFiles.md)
there:
```
$ cd $PROJECT_DIR
$ capstan package init --name {name} --title {title} --author {author}
$ capstan runtime init --runtime {runtime}
# edit meta/run.yaml to match your application structure
```
Being in project root directory, then use Capstan command to create unikernel
(consult [CLI Reference](Documentation/generated/CLI.md) for a list of available arguments):
```
$ capstan package compose {unikernel-name}
```
At this point, you have your unikernel built. It contains all your project files plus all the
precompiled artefacts that you asked for. In other words, the unikernel contains everything and is
ready to be started! As you might have expected, there is Capstan command to run unikernel for you
(using KVM/QEMU hipervisor):
```
$ capstan run {unikernel-name}
```
Congratulations, your unikernel is up-and-running! Press CTRL + C to stop it.

## Documentation

* [Step-by-step Capstan Installation Guide](Documentation/Installation.md)
* [Running My First Application Inside Unikernel](Documentation/WalkthroughNodeJS.md)
* [Configuration Files](Documentation/ConfigurationFiles.md)
* [CLI Reference](Documentation/generated/CLI.md)
* [Under the Hood](Documentation/UnderTheHood.md)

## License
Capstan is distributed under the 3-clause BSD license.

## Acknowledgements
This code has been developed within the [MIKELANGELO project](https://www.mikelangelo-project.eu)
(no. 645402), started in January 2015, and co-funded by the European Commission under the
H2020-ICT-07-2014: Advanced Cloud Infrastructures and Services programme.
