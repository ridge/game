//+build game

package main

import (
	"github.com/ridge/game/task"
)

var Default = FooBar

func WrongSignature(i int) {
}

func FooBar(ctx task.Context) {
	ctx.Dep(WrongSignature)
}
