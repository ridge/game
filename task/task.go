package task

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

// Span is a single span of task execution, either computation or waiting for subtasks
type Span struct {
	Start time.Time
	End   time.Time

	Subtasks []*Task
}

// Task contains information about single execution of a task
type Task struct {
	ID       int
	Runnable Runnable

	once     sync.Once
	reporter Reporter

	// Fields below are filled during t.Run()
	Spans []Span
	Error error // nil if the task succeeded
}

// StringID formats task ID
func (t *Task) StringID() string {
	return fmt.Sprintf("#%04d", t.ID)
}

// Name formats task name
func (t *Task) Name() string {
	switch r := t.Runnable.(type) {
	case identifiable:
		return r.Identify()
	case fmt.Stringer:
		return r.String()
	default:
		return fmt.Sprintf("%#v", r)
	}
}

func (t *Task) String() string {
	return fmt.Sprintf("#%04d %s", t.ID, t.Name())
}

func (t *Task) Start() time.Time {
	return t.Spans[0].Start
}

// End is the time the task ended
func (t *Task) End() time.Time {
	return t.Spans[len(t.Spans)-1].End
}

// Duration returns duration of the task
func (t *Task) Duration() time.Duration {
	return t.End().Sub(t.Spans[0].Start)
}

// SelfDuration returns duration of task computation without subtasks
func (t *Task) SelfDuration() time.Duration {
	var d time.Duration
	for _, s := range t.Spans {
		if len(s.Subtasks) == 0 {
			d += s.End.Sub(s.Start)
		}
	}
	return d
}

type contextKey string

const (
	taskContextKey = contextKey("game.task")
)

type flushWriter interface {
	io.Writer
	Flush()
}

type taskContext struct {
	task *Task

	nextSpanStart time.Time
	stdout        flushWriter
	stderr        flushWriter
}

func taskCtx(ctx Context) *taskContext {
	return ctx.Value(taskContextKey).(*taskContext)
}

// Stdout returns a stdout writer associated with the current task
func Stdout(ctx Context) io.Writer {
	return taskCtx(ctx).stdout
}

// Stderr returns a stderr writer associated with the current task
func Stderr(ctx Context) io.Writer {
	return taskCtx(ctx).stderr
}

func (tc *taskContext) closeSpan(subtasks []*Task) {
	endTime := time.Now()
	tc.task.Spans = append(tc.task.Spans, Span{Start: tc.nextSpanStart, End: endTime, Subtasks: subtasks})
	tc.nextSpanStart = endTime
}

func (t *Task) run(ctx Context) {
	t.reporter.Started(t)

	stdout, stderr := newStreamLineWriters(t, t.reporter)

	tc := &taskContext{
		task:          t,
		nextSpanStart: time.Now(),
		stdout:        stdout,
		stderr:        stderr,
	}
	ctx.Context = context.WithValue(ctx.Context, taskContextKey, tc)

	defer func() {
		tc.closeSpan(nil)

		if e := recover(); e != nil {
			if err, ok := e.(error); ok {
				t.Error = err
			} else {
				t.Error = fmt.Errorf("%v", e)
			}
		}

		t.reporter.Finished(t)
	}()

	t.Runnable.Run(ctx)
}

// Run runs the task
func (t *Task) Run(ctx Context) {
	t.once.Do(func() {
		t.run(ctx)
	})
}

// SubtasksFailure is an error raised if any subtask of a task has failed
type SubtasksFailure []*Task

func plural(name string, count int) string {
	if count == 1 {
		return name
	}

	return name + "s"
}

func (st SubtasksFailure) Error() string {
	ids := make([]string, 0, len(st))
	for _, subtask := range st {
		ids = append(ids, subtask.String())
	}
	return fmt.Sprintf("Failed %s: %s",
		plural("subtask", len(st)),
		strings.Join(ids, ", "))
}

func reportFailures(tc *taskContext, failures []*Task) {
	sort.Slice(failures, func(i, j int) bool {
		return failures[i].ID < failures[j].ID
	})
	tc.task.Error = SubtasksFailure(failures)
	panic(tc.task.Error)
}

// runSubtasks runs given tasks as subtasks of the task in the context in
// parallel. All tasks are allowed to finish even if some of them error out, and
// errors from all subtasks are collected.
func runSubtasks(ctx Context, subtasks []*Task) {
	tc := taskCtx(ctx)
	tc.closeSpan(nil)

	finishedCh := make(chan *Task, len(subtasks))

	tc.task.reporter.Dependencies(tc.task, subtasks, false)

	for _, subtask := range subtasks {
		go func(subtask *Task) {
			subtask.Run(ctx)
			finishedCh <- subtask
		}(subtask)
	}

	var f []*Task

	for i := 0; i < len(subtasks); i++ {
		subtask := <-finishedCh
		if subtask.Error != nil {
			f = append(f, subtask)
		}
	}

	tc.closeSpan(subtasks)

	if len(f) > 0 {
		reportFailures(tc, f)
	}
}

func runSubtasksSequential(ctx Context, subtasks []*Task) {
	tc := taskCtx(ctx)
	tc.closeSpan(nil)

	tc.task.reporter.Dependencies(tc.task, subtasks, true)

	var f []*Task

	for _, subtask := range subtasks {
		subtask.Run(ctx)
		if subtask.Error != nil {
			f = append(f, subtask)
		}
	}

	tc.closeSpan(subtasks)

	if len(f) > 0 {
		reportFailures(tc, f)
	}
}
