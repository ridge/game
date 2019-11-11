// +build mage

// Compiled package description.
package main

import (
	"context"
	"log"
	"time"

	"github.com/magefile/mage/mg"
)

var Default = Deploy

// This is very verbose.
func TestVerbose() {
	log.Println("hi!")
}

// This is the synopsis for Deploy. This part shouldn't show up.
func Deploy() {
	mg.Deps(f)
}

// Sleep sleeps 5 seconds.
func Sleep(ctx context.Context) {
	select {
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
		panic(ctx.Err())
	}
}

func f() {
	log.Println("i am independent -- not")
}
