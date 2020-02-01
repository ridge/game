// +build game

package main

import (
	"fmt"
	"log"

	"github.com/ridge/game/mg"
	"github.com/ridge/game/task"
)

// This should work as a default - even if it's in a different file
var Default = ReturnsNilError

// this should not be a target because it returns a string
func ReturnsString() string {
	fmt.Println("more stuff")
	return ""
}

func TestVerbose() {
	log.Println("hi!")
}

func ReturnsVoid(ctx task.Context) {
	mg.CtxDeps(ctx, f)
}

func f() {}
