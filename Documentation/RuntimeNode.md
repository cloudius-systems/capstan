# Runtime `node`
This document describes how to write a valid `meta/run.yaml` configuration file
for running **Node.js** application. Please note that you needn't require Node
MPM package manually since Capstan will require require following package automatically:

```
- node-4.4.5
```

## Interactive node interpreter
Following configuration can be used to run interactive Node.js interpreter inside OSv:

```yaml
# meta/run.yaml

runtime: node

config_set:
  interpreter:
    shell: true
```

Example:

```bash
$ capstan package compose demo
$ capstan run demo --boot interpreter
Command line will be set based on --boot parameter
Created instance: demo
Setting cmdline: runscript /run/interpreter
OSv v0.24-434-gf4d1dfb
eth0: 192.168.122.15
> console.log("Hello World from Node.js interactive interpreter!")
Hello World from Node.js interactive interpreter!
undefined
>
```

## Node.js script
Following configuration can be used to run Node.js script inside OSv:

```yaml
# meta/run.yaml

runtime: node

config_set:
  hello:
    main: /greeting.js
    args:
      - "Johnny"
```
Note that /greeting.js script mentioned in the snippet above is a simple script that we've
implemented for the sake of demo. It prints node arguments to the console and then
exits.

Example:

```bash
$ capstan package compose demo
$ capstan run demo --boot hello
Command line will be set based on --boot parameter
Created instance: demo
Setting cmdline: runscript /run/hello
OSv v0.24-434-gf4d1dfb
eth0: 192.168.122.15
Hello to:
- Johnny
```
