# panicmonitor

> A program that monitors Go programs and do something when they crash.

Inspired by [panicwrap](https://github.com/mitchellh/panicwrap).

## Installation

```sh
go install -u github.com/lujjjh/panicmonitor/cmd/panicmonitor
```

## Configuration

Create a configuration file wherever you want, for example, `/etc/panicmonitor.toml`.

```toml
[dingtalk]
webhook = 'https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN_HERE'
```

## Usage

```sh
panicmonitor /etc/panicmonitor.toml /path/to/your/executable arguments
```
