//+build game

package main

import (
	"fmt"

	"github.com/ridge/game/mg"
	"github.com/ridge/game/task"
)

var Default = NS.BareCtx

func TestNamespaceDep(ctx task.Context) {
	ctx.Dep(NS.BareCtx)
}

type NS mg.Namespace

func (NS) BareCtx(ctx task.Context) {
	fmt.Println("hi!")
}
