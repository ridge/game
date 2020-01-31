//+build game

package main

import "github.com/ridge/game/game/testdata/mixed_lib_files/subdir"

func Build() {
	subdir.Build()
}
