# Usage

To run OSv on default hypervisor which is QEMU/KVM, type:

```
$ capstan run cloudius/osv
```

To run OSv on VirtualBox, type:

```
$ capstan run -p vbox cloudius/osv
```

To port-forwarding OSv port 22 to Host port 10022, type:

```
$ capstan run -f "10022:22" cloudius/osv
```

To bridging OSv vNIC to Host bridge interface, type:

```
On Linux:
$ capstan run -n bridge cloudius/osv

On OS X with VirtualBox:
$ capstan run -n bridge -b <physical NIC name> cloudius/osv
```

To show a list of available remote images, type:

```
$ capstan search
```

To show a list of locally installed images, type:

```
$ capstan images
```

# Examples

Check out the following example projects to get you going:

* [Node.js](https://github.com/cloudius-systems/capstan-example-nodejs)
* [Java](https://github.com/cloudius-systems/capstan-example-java)
* [Clojure](https://github.com/cloudius-systems/capstan-example-clojure)
* [Linux binaries](https://github.com/cloudius-systems/capstan-example)

