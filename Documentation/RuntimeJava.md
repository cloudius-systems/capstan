# Runtime `java`
This document describes how to write a valid `meta/run.yaml` configuration file
for running **Java** application. Please note that you needn't require Java
MPM package manually since Capstan will require following package automatically:

```
- openjdk8-zulu-compact1
```

## Running .class
Following configuration can be used to run a javac-compiled Java application inside OSv:

```yaml
# meta/run.yaml

runtime: java

config_set:
  hello:
    main: MyClass
    args:
      - Johnny
```

Where the content of MyClass.java file is:

```java
public class MyClass
{
  public static void main(String[] args)
   {
     System.out.println("Hello:");
     for(String el : args){
       System.out.printf("- %s\n", el);
     }
   }
}
```
Please note that OSv doesn't really need the .java file, but only the .class file that was
produced like this:

```bash
$ javac MyClass.java
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
java.so: Starting JVM app using: io/osv/nonisolated/RunNonIsolatedJvmApp
java.so: Setting Java system classloader to NonIsolatingOsvSystemClassLoader
Hello:
- Johnny
```

## Running .jar
Following configuration can be used to run .jar file inside OSv:

```yaml
# meta/run.yaml

runtime: java

config_set:
  hello:
    main: /MyClass.jar
    args:
      - Johnny
```
The MyClass.jar was prepared out of the very same MyClass.java file as provided above by using the following
set of commands:

```bash
$ javac MyClass.java
$ jar cfe MyClass.jar MyClass MyClass.class
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
java.so: Starting JVM app using: io/osv/nonisolated/RunNonIsolatedJvmApp
java.so: Setting Java system classloader to NonIsolatingOsvSystemClassLoader
Hello:
- Johnny
```
