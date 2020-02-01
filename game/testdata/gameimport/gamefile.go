//+build game

package main

// important things to note:
// * these two packages have the same package name, so they'll conflict
// when imported.
// * one is imported with underscore and one is imported normally.
//
// they should still work normally as gameimports

import (
	"fmt"

	// game:import
	_ "github.com/ridge/game/game/testdata/gameimport/subdir1"
	// game:import zz
	game "github.com/ridge/game/game/testdata/gameimport/subdir2"
)

var Default = game.NS.Deploy2

func Root() {
	fmt.Println("root")
}
