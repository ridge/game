package game

import (
	"fmt"

	"github.com/ridge/game/mg"
)

// BuildSubdir Builds stuff.
func BuildSubdir() {
	fmt.Println("buildsubdir")
}

// NS is a namespace.
type NS mg.Namespace

// Deploy deploys stuff.
func (NS) Deploy() {
	fmt.Println("deploy")
}
