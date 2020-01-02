//+build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/task"
)

var Default = FooBar

func WrongSignature(i int) {
}

func FooBar(ctx task.Context) {
	mg.CtxDeps(ctx, WrongSignature)
}
