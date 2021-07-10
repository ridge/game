package toplevel

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ridge/game/task"
)

func formatLine(line task.LogLine) string {
	if line.Stream == task.StderrStream {
		return "E | " + line.Line
	}
	return "  | " + line.Line
}

type fileReporter struct {
	File *os.File
}

func (fr fileReporter) Started(t *task.Task) {
	fmt.Fprintf(fr.File, "%s STARTED %s\n", t.StringID(), t.Name())
}

func (fr fileReporter) Dependencies(dependent *task.Task, dependees []*task.Task, sequential bool) {
	s := []string{}
	for _, d := range dependees {
		s = append(s, d.String())
	}
	op := "DEPS"
	if sequential {
		op = "SEQDEPS"
	}
	fmt.Fprintf(fr.File, "%s %s %s -> %s\n", dependent.StringID(), op, dependent.Name(), strings.Join(s, ", "))
}

func (fr fileReporter) Finished(t *task.Task) {
	tag := "SUCCEEDED"
	if t.Error != nil {
		tag = "FAILED"
		msg := t.Error.Error()
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		for _, line := range strings.SplitAfter(msg, "\n") {
			if line == "" {
				continue
			}
			fr.OutputLine(t, t.End(), task.LogLine{Stream: task.StderrStream, Line: line})
		}
	}
	dur := t.Duration()
	self := t.SelfDuration()
	fmt.Fprintf(fr.File, "%s %s %s time=%.02fs, self=%.02fs, subtasks=%.02fs\n",
		t.StringID(), tag, t.Name(), dur.Seconds(), self.Seconds(), (dur - self).Seconds())
}

func (fr fileReporter) OutputLine(t *task.Task, time time.Time, line task.LogLine) {
	fmt.Fprintf(fr.File, "%s %s", t.StringID(), formatLine(line))
}
