package task

import (
	"context"
	"sync"
	"time"
)

// Stream is the standard I/O stream type
type Stream int

const (
	// StdoutStream is a marker of stdout stream
	StdoutStream Stream = 1
	// StderrStream is a marker of stderr stream
	StderrStream Stream = 2
)

// Reporter reports events
type Reporter interface {
	Started(t *Task)
	Finished(t *Task)
	Dependencies(dependent *Task, dependees []*Task)
	OutputLine(t *Task, time time.Time, stream Stream, line string)
}

// Registry registers runnables for execution
type Registry struct {
	reporter Reporter

	mu     sync.Mutex
	tasks  map[string]*Task
	nextID int
}

// Runnable is a named piece of runnable code
type Runnable interface {
	Name() string
	Run(ctx context.Context) error
}

// Register registers runnable as a task
func (r *Registry) Register(runnables []Runnable) []*Task {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]*Task, 0, len(runnables))
	for _, runnable := range runnables {
		name := runnable.Name()

		if _, exists := r.tasks[name]; !exists {
			r.tasks[name] = &Task{
				ID:       r.nextID,
				Runnable: runnable,
				reporter: r.reporter,
			}
			r.nextID++
		}
		out = append(out, r.tasks[name])
	}
	return out
}

// All is a registry of all tasks
var All = Registry{
	tasks: map[string]*Task{},
}

// SetReporter sets the reporter
func SetReporter(reporter Reporter) {
	All.reporter = reporter
}
