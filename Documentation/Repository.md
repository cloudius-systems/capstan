# Capstan Repository
Capstan tool downloads base unikernel and precompiled packages from OSv Github releases repository
 (the default) or publicly available remote repository in AWS S3.
If S3 mode selected (--s3), following S3 repository is used:

```
https://mikelangelo-capstan.s3.amazonaws.com
```

There is nothing special about repository, it's just a bunch of directories and files that are made
available on the internet. No authentication is supported, just a simple HTTP/HTTPS download.

## OSv Github Repository
By default capstan pulls OSv kernel and any required packages from [OSv Github repository](https://github.com/cloudius-systems/osv/releases).
The global parameter ```--releaseTag``` (or ```-rt```) can be used to override default behavior and make
capstan pull artifacts published for specific release (for example ```--rt v0.51.0```) or the latest release (```--rt latest```).
By default capstan would pull first found artifact from the list of assets published for all releases (```--rt any```).

## S3 Repository Structure
One can use global ```--s3``` parameter to make capstan pull kernel and packages from repository in AWS S3.
The default public repository created as part of MIKELANGELO project is structured as follows:

```
mike
 |- osv-loader                  #
 |   |- index.yaml              # base unikernel (path is hard-coded in Capstan)
 |   |- osv-loader.qemu.gz      #
packages
 |- osv.bootstrap.mpm           # actual package
 |- osv.bootstrap.yaml          # metadata describing package
 |- erlang-7.0.mpm
 |- erlang-7.0.yaml
 |- ...
```

As shown above, there are two root directories in the repository: `mike` and `packages`.

### `mike` root directory
This directory contains base unikernel that Capstan builds your own unikernel upon. The
"recipe" to prepare the two files is best shown [here](https://github.com/mikelangelo-project/capstan-packages/blob/master/docker_files/capstan-packages.py#L226-L259). To prepare the `osv-loader.qemu.gz` file we simply checkout
latest [OSv master](https://github.com/cloudius-systems/osv) and build it. Then we take the
`$OSV_DIR/build/last/loader.img` file and tar.gz compress it into `osv-loader.qemu.gz`.

### `packages` root directory
This direcotry contains all the Capstan packages. Each package is stored in two files that
only differ in suffix: `{package-name}.mpm` and `{package-name}.yaml`. The former is actually
a tar.gz file containting all the package files, but please make use of `capstan package build`
command to compress it, as can be seen in [this](https://github.com/mikelangelo-project/capstan-packages/blob/master/docker_files/capstan-packages.py#L391-L425) "recipe". The latter is a simple metafile in yaml format.

## Hosting your own repository
It should be very simple to host your own Capstan repository, just make sure that you maintain
the directory structure as described above.

Please consult [this](./Installation.md#2-using-configuration-file) file to learn how to tell Capstan
what remote repository to connect to.
