# Attaching volumes
Capstan supports attaching any number of pre-prepared volumes to the instance. Currenly, you need to prepare
a volume manually i.e. with external tools, and then export it into a file on your host. During the `capstan run`
you then provide path to this file together with some metadata.

NOTE: volumes are only supported for QEMU hipervisor at the moment.

## Creating a new volume
You can create a new volume using qemu-img tool:

```bash
$ qemu-img create -f qcow2 ./volume.img 1G
```

## Attaching volumes
Tell Capstan where your volume is and it will attach it to the instance on run:

```bash
$ capstan run demo --volume ./volume.img
```

The volume will get attached as `/dev/vblk1` into the unikernel. The `--volume` argument can be repeated
to attach multiple volumes:

```bash
$ capstan run demo --volume ./volume1.img --volume ./volume2.img --volume ./volume3.img
```

Volumes will get attached as `/dev/vblk1`, `/dev/vblk2`, `/dev/vblk3` in the same order as provided (left-to-right).

### Volume metadata
You can provide volume metadata for Capstan to be able to attach it as desired:

```bash
$ capstan run demo --volume ./volume.img:format=qcow2:aio=threads
```

| KEY | DEFAULT VALUE | VALUES |
|-------|---------------------|------------|
| format | raw | raw, qcow2,vdi,vmdk|
| aio | native | native/threads|
| cache | none | none/writeback/writethrough/directsync/unsafe|

*Consult [QEMU documentation](https://qemu.weilnetz.de/doc/qemu-doc.html#Block-device-options) for more details.*
