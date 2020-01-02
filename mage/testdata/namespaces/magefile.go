//+build mage

package main

import (
	"fmt"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/task"
)

var Default = NS.BareCtx

func TestNamespaceDep(ctx task.Context) {
	mg.CtxDeps(ctx, NS.BareCtx)
}

type NS mg.Namespace

func (NS) BareCtx(ctx task.Context) {
	fmt.Println("hi!")
}
