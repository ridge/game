//go:build !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)

package tty

import (
	"fmt"

	"github.com/ridge/game/task"
)

func NewReporter() (task.Reporter, error) {
	return nil, fmt.Errorf("TTY reporter is only available under Unix")
}
