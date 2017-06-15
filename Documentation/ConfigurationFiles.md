# Capstan Configuration Files
You must provide two configuration files to run your application using Capstan:
<pre>
{application-root}
└─── meta
       |- <b>package.yaml</b>
       |- <b>run.yaml</b>
| ... (application files and directories)
</pre>

Both files must be placed inside directory named `meta/` that you create directly inside
your project's root directory. The [package.yaml](#metapackageyaml) file tells Capstan how to
properly *create* unikernel i.e. what precompiled packages to put in it besides your application
files. And the [run.yaml](#metarunyaml) file gives Capstan all the neccessary information about your
application (the runtime, the main file, ...) to *run* it.

## meta/package.yaml
Below please find sample content of meta/package.yaml configuration file:
```yaml
name: my-super-application
title: DEMO App
author: lemmy (lemmy@email.com)
require:
    - osv.cli
```
The first three parameters are simple metadata and are used later if you decide to publish your
application as a package to make it available for everyone. Be careful, though, that you pick unique
name for your package to avoid collisions with other users.

The most interesting attribute is the one named `require`. This is where you list all the packages
that you would like to have them included in your final unikernel. In this example we've listed only
one, `osv.cli`. You can omit the whole attribute if you don't need any
precompiled packages. A list of all packages in the remote repository can be obtained by executing:
```bash
$ capstan package search
Name                                               Description                       Version
app.hadoop-hdfs             Hadoop HDFS                       2.7.2
app.hello-node              NodeJS-4.4.5                      4.4.5
app.mysql-5.6.21            MySQL-5.6.21                      5.6.21
app.node-4.4.5              NodeJS-4.4.5                      4.4.5
erlang                      Erlang                            18.0
ompi                        Open MPI                          1.10
openfoam.core               OpenFOAM Core                     2.4.0
openfoam.pimplefoam         OpenFOAM pimpleFoam               2.4.0
openfoam.pisofoam           OpenFOAM pisoFoam                 2.4.0
openfoam.poroussimplefoam   OpenFOAM porousSimpleFoam         2.4.0
openfoam.potentialfoam      OpenFOAM potentialFoam            2.4.0
openfoam.rhoporoussimplefoam OpenFOAM rhoPorousSimpleFoam     2.4.0
openfoam.rhosimplefoam      OpenFOAM rhoSimpleFoam            2.4.0
openfoam.simplefoam         OpenFOAM simpleFoam               2.4.0
osv.bootstrap               OSv Bootstrap                     v0.24-216-g1cf8972
osv.cli                     OSv Command Line Interface        v0.24-216-g1cf8972
osv.cloud-init              cloud-init                        v0.24-216-g1cf8972
osv.httpserver              OSv HTTP REST Server              v0.24-216-g1cf8972
osv.java                    Java JRE 1.7.0                    v0.24-216-g1cf8972
osv.nfs                     OSv NFS Client Tools              v0.24-216-g1cf8972
```
Then to download desired package into your local repository execute:
```bash
$ capstan package pull {package-name}
```
Alternatively, you can make use of `--pull-missing` flag when composing unikernel.

Please note that packages are copied to the unikernel in the same order that you specify here in
meta/package.yaml file. So if two packages contain file with same name and inside same folder path,
then the one that was copied last will remain. Only after all the packages are copied to the unikernel,
your application files are copied too. So your application can never get overwritten. To verify the
final content one can execute:
```bash
$ capstan package collect
```
A folder `mpm-pkg` appears containing exact content as it will be baked into unikernel during compose.


## meta/run.yaml
Content of run.yaml file depends on runtime that this package is about to use. File is structured
as follows:
```yaml
runtime: {runtime-name}

config_set:
   {configuration-name}:
      {configuration}
   {configuration-name}:
      {configuration}
   ...
config_set_default: {configuration-name}
```
Key `runtime` defines what runtime should be used to run our application. Within key `config_set`
we then define one or more named configurations. Configuration name is arbitrary string, while
configuration itself is a set of key-value pairs required by selected runtime above.
Key `config_set_default` defines what configuration set should be run by default i.e. if not otherwise
specified via command-line parameters of `capstan package compose` command. It can be omitted when
only one configuration set exists (it then becomes the default one).

A list of all runtimes can be obtained by executing:
```
$ capstan runtime list

RUNTIME    DESCRIPTION                                 DEPENDENCIES
native     Run arbitrary command inside OSv            []
node       Run JavaScript NodeJS 4.4.5 application     [app.node-4.4.5]
java       Run Java 1.7.0 application                  [osv.java]
```
And then Capstan can tell us what settings are supported for each runtime. For example, for NodeJS
one can view expected content of the run.yaml file by executing:
```
$ capstan runtime preview -r node

--------- meta/run.yaml ---------
runtime: node

config_set:

   ################################################################
   ### This is one configuration set (feel free to rename it).  ###
   ################################################################
   myconfig1:
      # REQUIRED
      # Filepath of the NodeJS entrypoint (where server is defined).
      # Note that package root will correspond to filesystem root (/) in OSv image.
      # Example value: /server.js
      main: <filepath>

      # OPTIONAL
      # Environment variables.
      # A map of environment variables to be set when unikernel is run.
      # Example value:  env:
      #                    PORT: 8000
      #                    HOSTNAME: www.myserver.org
      env:
         <key>: <value>

   # Add as many named configurations as you need

# OPTIONAL
# What config_set should be used as default.
# This value can be overwritten with --runconfig argument.
config_set_default: myconfig1
---------------------------------
```
Lets sum up. To prepare appropriate run.yaml file, we must first select one of the supported runtimes.
If we are about to be running NodeJS application, then we opt-in to use runtime named *node*. We get
the details on how to prepare run.yaml for *node* by using Capstan command.


## Automatic generation of configuration files
You can create configuration files manually or generate them using Capstan. The latter option does
not only create empty files; Capstan pre-fills them with default values and detailed self-description
in form of yaml comments. You are therefore advised to use Capstan to initialize configuration files
for you.

To initialize meta/package.yaml file, use:
```
$ capstan package init \
   --name "my-super-application" \
   --title "DEMO App" \
   --author "lemmy (lemmy@email.com)" \
   --require osv.cli
```
This will create a meta subdirectory and ``meta/package.yaml`` file with the
given content. Then to initialize meta/run.yaml file, use:
```
$ capstan runtime init -r {runtime-name}
```
This will create ``meta/run.yaml`` file with documentation for the selected runtime.




