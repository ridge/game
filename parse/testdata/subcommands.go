// +build game

package main

import "github.com/ridge/game/mg"

type Build mg.Namespace

func (Build) Foobar() error {
	// do your foobar build
	return nil
}

func (Build) Baz() {
	// do your baz build
}

type Init mg.Namespace

func (Init) Foobar() error {
	// do your foobar defined in init namespace
	return nil
}
