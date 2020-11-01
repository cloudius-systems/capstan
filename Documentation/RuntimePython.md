# Runtime `python`
This document describes how to write a valid `meta/run.yaml` configuration file
for running **Python** applications.

# Newer `native` method
Require python:
```yaml
require:
        - osv.python3x
```

And set `bootcmd` to `python3`:
```yaml
# meta/run.yaml
runtime: native

config_set: 
   default:
      bootcmd: python3
```

# Deprecated `python` method
Note that you needn't require Python MPM package manually since Capstan will require following package automatically:
```
- python-2.7
```

The following configuration runs an interactive Python interpreter using the python runtime:
```yaml
# meta/run.yaml

runtime: python

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
OSv v0.24-448-g829bf76
eth0: 192.168.122.15
Python 2.7.13+ (heads/2.7:883520a, Aug 17 2017, 08:15:22)
[GCC 4.8.4] on linux2
Type "help", "copyright", "credits" or "license" for more information.
>>>
```

## Python script
Following configuration can be used to run Python script inside OSv:

```yaml
# meta/run.yaml

runtime: python

config_set:
  hello:
    main: /script.py
    args:
      - Johnny
```
Note that /script.py script mentioned in the snippet above is a simple script that we've
implemented for the sake of demo. It prints python arguments to the console:

```python
import sys

print 'Hello:'
for el in sys.argv[1:]:
  print '- %s' % el
```

Example:

```bash
$ capstan package compose demo
$ capstan run demo --boot hello
Command line will be set based on --boot parameter
Created instance: demo
Setting cmdline: runscript /run/hello
OSv v0.24-448-g829bf76
eth0: 192.168.122.15
Hello:
- Johnny
```
