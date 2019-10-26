# Building from Source

## Requirements

- [Go](https://golang.org/dl/) version 1.11 or later.
- [Node.js](https://nodejs.org/en/download/) LTS version.
- A Unix shell environment.
- (Optional) docker.

## Steps

- Use `git` to clone this repository, or download the ZIP file from [here](https://github.com/defenxor/dsiem/archive/master.zip) and extract it to a working directory.

- Make sure you have both `go` and `npm` in `$PATH`. These commands should work from all location:
  
  ```shell
  $ go version
  $ npm -v
  ```

- Open terminal and `cd` to Dsiem working directory.

- To build all Dsiem commands (linux version):

  ```shell
  $ ./scripts/gobuild-cmd.sh
  ```
  or to build a single command, for example `dpluger`:

  ```shell
  $ ./scripts/gobuild-cmd.sh ./cmd/dpluger
  ```
  The result will all be located in the working directory.

- To also build the commands for Windows and Darwin version:
  ```shell
  $ ./scripts/gobuild-cmd-release.sh
  ```
  The result will be several zip files located in temp/release/latest/ directory.

- To build the web UI:
  
  ```shell
  $ ./scripts/ngbuild-web.sh
  ```
  The result will be in `./web/dist` directory.

- To build the docker image:

  ```shell
  $ ./scripts/dockerbuild-dev.sh
  ```
  The result will be an image named `defenxor/dsiem-devel`.

