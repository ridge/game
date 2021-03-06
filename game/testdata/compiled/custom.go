// +build game

// Compiled package description.
package main

import (
	"log"
	"time"

	"github.com/ridge/game/task"
)

var Default = Deploy

// This is very verbose.
func TestVerbose(ctx task.Context) {
	log.Println("hi!")
}

// This is the synopsis for Deploy. This part shouldn't show up.
func Deploy(ctx task.Context) {
	ctx.Dep(f)
}

// Sleep sleeps 5 seconds.
func Sleep(ctx task.Context) {
	select {
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
		panic(ctx.Err())
	}
}

func f() {
	log.Println("i am independent -- not")
}
