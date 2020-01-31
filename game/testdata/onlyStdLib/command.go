// +build game

package main

import (
	"fmt"
	"log"

	"github.com/ridge/game/mg"
	"github.com/ridge/game/task"
)

var Default = SomePig

// this should not be a target because it returns a string
func ReturnsString() string {
	fmt.Println("more stuff")
	return ""
}

func TestVerbose() {
	log.Println("hi!")
}

// This is the synopsis for SomePig.  There's more data that won't show up.
func SomePig(ctx task.Context) {
	mg.CtxDeps(ctx, f)
}

func f() {}
