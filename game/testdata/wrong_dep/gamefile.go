//+build game

package main

import (
	"github.com/ridge/game/mg"
	"github.com/ridge/game/task"
)

var Default = FooBar

func WrongSignature(i int) {
}

func FooBar(ctx task.Context) {
	mg.CtxDeps(ctx, WrongSignature)
}
