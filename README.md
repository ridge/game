# Game

Game is a Make-like build tool using Go.

You describe build tasks in Go, and Game runs them.

## Installation

`go get github.com/ridge/game`

`game` binary will be installed in your `$GOPATH/bin` directory.

## Discussion

Join the `#game` channel on [gophers slack](https://gophers.slack.com/) or
[gamefile Google Group](https://groups.google.com/forum/#!forum/gamefile).

## Documentation

Documentation website TBD, see [this directory](site/content) in meantime.

See [godoc](https://godoc.org/github.com/ridge/game/game) for how to use Game as
a library.

## Why?

Mage has a [great answer](https://magefile.org/#why).

## How is Game different from Mage?

Game is indeed a a fork of [Mage](https://magefile.org).

Mage prides itself on having clean and readable build files. Game trades a bit
of readability for features.

- New features of Game:
  - parameterized dependencies
  - task time tracing
  - per-task output capture (output from parallel tasks is not intermixed)
  - shorthand `ctx.Dep` instead of `mg.CtxDeps(ctx, ...)`

- Game drops several features of Mage:
  - Alias targets
  - GOPATH environment support
  - Shorthand tasks definition without `Context` parameter

## License

Game is licensed under terms of [Apache 2.0 license](LICENSE).
