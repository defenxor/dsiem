# Dsiem Demo — VM version

This directory holds the files to build virtual machine version of [Dsiem demo](../). This is provided as an alternative for those who don't want to install Docker toolchain on their main OS.

## Requirements

- [VirtualBox](https://www.virtualbox.org/wiki/Downloads).
- [Vagrant](https://www.vagrantup.com/downloads.html).
- 4 CPU cores and 6 GB of RAM dedicated to the VM.
- Disk space from 1 GB (build only, e.g. for exporting OVA) to 32 GB depending on VM selection and usage.

Vagrant also support other [hypervisors](https://www.vagrantup.com/docs/providers/), but the build steps here have only been tested on VirtualBox.

## Building

- Clone this repo using `git` or download the [zipped version](https://github.com/defenxor/dsiem/archive/master.zip).
- Execute `build-vm.sh` script:

```shell
$ cd demo/vagrant
$ ./ build-vm.sh
```

The script will first ask if it should build an Alpine or an Ubuntu-based virtual machine. After that, build process should continue and be completed within a few minutes.

## Usage

Dsiem demo VM is meant to be accessed only by a local user using a web browser. It isn't meant to be hosted on an open network, so by-default no traffic initiated from external host is allowed. The VM is configured with two virtual network adapters:

- A NAT network adapter for vagrant SSH channel, and for downloading docker images from the Internet.
- A host-only network adapter that allows user access to the demo web interface.

You can use and interact with the VM directly through VirtualBox GUI: login on the console as `demo` user to start the demo, or `root` user to perform system administration. Both accounts have no password configured.

Or you can also skip VirtualBox GUI and just use `vagrant` to start the demo:

```shell
$ cd demo/vagrant/alpine    <-- or ubuntu, depending on which VM was built
$ vagrant up && vagrant ssh -c 'su - demo'
```

## Updating

The VM will try to update Dsiem source files and tools releases from github.com, and pull updated docker images every time `demo` user logs in. This is meant to help in keeping the VM content updated with the latest version of the demo available. If this is undesirable at some point, then just disconnect the default NAT network before logging in as `demo` user.

## Clean up

Remove the VM using VirtualBox GUI, or by running `vagrant destroy` in the respective VM directory.
