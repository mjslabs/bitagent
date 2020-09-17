bitagent
===

[![Build Status][travis-badge]][travis]
[![Go Report Card][goreport-badge]][goreport]
[![Test Coverage][coverage]][codeclimate]
[![FOSSA Status][fossa-badge]][fossa]

[travis-badge]: https://travis-ci.org/mjslabs/bitagent.svg?branch=master
[travis]: https://travis-ci.org/mjslabs/bitagent
[goreport-badge]: https://goreportcard.com/badge/github.com/mjslabs/bitagent
[goreport]: https://goreportcard.com/report/github.com/mjslabs/bitagent
[coverage]: https://api.codeclimate.com/v1/badges/221f0a7b9a102e962743/test_coverage
[codeclimate]: https://codeclimate.com/github/mjslabs/bitagent/test_coverage
[fossa-badge]: https://app.fossa.com/api/projects/git%2Bgithub.com%2Fmjslabs%2Fbitagent.svg?type=shield
[fossa]: https://app.fossa.com/projects/git%2Bgithub.com%2Fmjslabs%2Fbitagent?ref=badge_shield

bitagent is a small service to help you share secrets between processes. Run
locally, each instance of bitagent listens on a Unix socket and is able to store
one secret. Instead of storing a session key or password in a file, store it in
a bitagent process. This keeps it off your disk, out of `ps` output, and
eliminates accidental password leaks to your shell's history file.

Why
---

Occasionally there are passwords or session keys that need to be accessed from
multiple processes and login sessions but that should not be written to disk.
The inspiration for this is Bitwarden's CLI, which requires a session key to be
passed or set in the environment. Sharing this session key between shell
sessions (or when using
[ansible](https://github.com/c0sco/ansible-modules-bitwarden)) can be
cumbersome.  

Each bitagent process is capable of storing only one secret. This keeps the code
simple, which helps keep it performant and reduces the chance of errors.

Installation
---

To install bitagent, use the standard `go install` process.
[Go 1.11+](https://golang.org/) is required due to the use of Go modules.

```bash
git clone https://github.com/mjslabs/bitagent.git && cd bitagent
go install
```

Usage
---

The easiest way to work with bitagent is by making a wrapper script for your
use case. See [examples](examples), which includes such a script for use with
the Bitwarden CLI. Below are the instructions for working with bitagent
manually, or when creating your own wrapper script. If you're only interested in
the Bitwarden use case, see the comments at the top of the
[bw](examples/bw) example file.

If you're interested in running bitagent manually, or creating your own wrapper,
read on.

First, launch bitagent using your system's preferred method of backgrounding a
process, e.g.

```bash
${GOBIN}/bitagent & disown
```

By default bitagent will create `~/.bitagent.sock` for communication. You can
specify an alternative location for the socket by passing it as the one argument
to bitagent.  

To store a secret, send a `P` command. Here's an example using netcat.

```bash
echo "Pmysecret" | nc -U ~/.bitagent.sock -N
```

To retrieve the secret, use `G`.

```bash
echo "G" | nc -U ~/.bitagent.sock -N
```

Full example showing the storage of the string `mysecret`, then retrieving it.

```shell
$ echo Pmysecret | nc -U ~/.bitagent.sock -N
$ echo G | nc -U ~/.bitagent.sock -N
mysecret
```

Your wrapper script should understand the output of the command you're proxying,
parsing the output for whatever token you're looking to store, then use `nc` or
something similar to store and retrieve the secret as needed.

Caveats
---

bitagent uses [memguard](https://github.com/awnumar/memguard), which attempts to
stop the part of bitagent's memory that is holding a secret from being paged
out or included in core dumps. This has not been fully vetted by the authors of
bitagent.  

bitagent defaults to storing up to a 256 byte secret. This is tunable at the
top of [bitagent.go](cmd/bitagent.go). This should be made dynamic.

The only thing stopping someone from accessing your secret in bitagent is the
permissions on the socket file. These default to a sane value, but there are
no guarantees that this is the best practice for all environments. You should
only run bitagent on trusted machines. This is similar to how you would treat
your SSH private key file.
