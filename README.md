[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/ridge/game/)
[![Build Status](https://travis-ci.com/ridge/game.svg?branch=master)](https://travis-ci.com/ridge/game/)

# Game

Game is a Make-like build tool using Go.

You describe build tasks in Go, and Game runs them.

## Installation

`go get github.com/ridge/game`

`game` binary will be installed to your `$GOPATH/bin` directory.

## Discussion

Join the `#game` channel on [gophers slack](https://gophers.slack.com/) or
[gamefile Google Group](https://groups.google.com/forum/#!forum/gamefile).

## Documentation

Documentation website TBD, see [this directory](site/content) in meantime.

## Why?

Mage has a [great answer](https://magefile.org/#why).

## How is Game different from Mage?

Game is indeed a a fork of [Mage](https://magefile.org).

Mage prides itself on having clean and readable build files. Game trades a bit
of readability for features.

- New features of Game:
  - parameterized dependencies
  - top-level targets for variables implementing task.Runnable
  - task time tracing
  - per-task output capture (output from parallel tasks is not intermixed)
  - shorthand `ctx.Dep` instead of `mg.CtxDeps(ctx, ...)`
  - Functions, variables and methods named `All` are defaults.
    E.g. a method declaration `func (Hey) All(ctx task.Context) {}`
	produces a task named `hey`, not `hey:all`.

- Game drops several features of Mage:
  - GOPATH environment support
  - Shorthand tasks definition without `Context` parameter

## License

Game is licensed under terms of [Apache 2.0 license](LICENSE).
