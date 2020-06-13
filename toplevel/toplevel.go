package toplevel

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ridge/game/task"
)

func formatLine(line task.LogLine) string {
	if line.Stream == task.StderrStream {
		return "E | " + line.Line
	}
	return "  | " + line.Line
}

const maxTailLines = 200

func taskTail(t *task.Task) string {
	b := strings.Builder{}

	start := 0
	if len(t.Output) > maxTailLines {
		b.WriteString("<truncated, see logs above>\n")
		start = len(t.Output) - maxTailLines
	}
	for i := start; i < len(t.Output); i++ {
		b.WriteString(formatLine(t.Output[i]))
	}
	return b.String()
}

type consoleReporter struct {
}

func (consoleReporter) Started(t *task.Task) {
	fmt.Printf("%s STARTED %s\n", t.StringID(), t.Name())
}

func (consoleReporter) Dependencies(dependent *task.Task, dependees []*task.Task, sequential bool) {
	s := []string{}
	for _, d := range dependees {
		s = append(s, d.String())
	}
	op := "DEPS"
	if sequential {
		op = "SEQDEPS"
	}
	fmt.Printf("%s %s %s -> %s\n", dependent.StringID(), op, dependent.Name(), strings.Join(s, ", "))
}

func (cr consoleReporter) Finished(t *task.Task) {
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
			cr.OutputLine(t, t.End(), task.LogLine{Stream: task.StderrStream, Line: line})
		}
	}
	dur := t.Duration()
	self := t.SelfDuration()
	fmt.Printf("%s %s %s time=%.02fs, self=%.02fs, subtasks=%.02fs\n",
		t.StringID(), tag, t.Name(), dur.Seconds(), self.Seconds(), (dur - self).Seconds())
}

func (consoleReporter) OutputLine(t *task.Task, time time.Time, line task.LogLine) {
	fmt.Printf("%s %s", t.StringID(), formatLine(line))
}

// Target is one build target
type Target struct {
	Name     string
	Fn       interface{}
	Synopsis string
	Comment  string
}

func plural(name string, count int) string {
	if count == 1 {
		return name
	}

	return name + "s"
}

func parseBool(env string) bool {
	val := os.Getenv(env)
	if val == "" {
		return false
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		log.Printf("warning: environment variable %s is not a valid bool value: %v", env, val)
		return false
	}
	return b
}

func parseDuration(env string) time.Duration {
	val := os.Getenv(env)
	if val == "" {
		return 0
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		log.Printf("warning: environment variable %s is not a valid duration value: %v", env, val)
		return 0
	}
	return d
}

func listTargets(targets []Target, defaultTarget string, desc string) {
	if desc != "" {
		fmt.Print(desc + "\n\n")
	}

	fmt.Println("Targets:")
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)
	for _, target := range targets {
		mark := ""
		if target.Name == defaultTarget {
			mark = "*"
		}
		fmt.Fprintf(w, "  %s%s\t%s\n", target.Name, mark, target.Synopsis)
	}
	w.Flush()
	if defaultTarget != "" {
		fmt.Println("\n* default target")
	}
	os.Exit(0)
}

func findTarget(haystack []Target, needle string) *Target {
	needle = strings.ToLower(needle)
	for _, target := range haystack {
		if strings.ToLower(target.Name) == needle {
			return &target
		}
	}
	return nil
}

func printMultilineIndented(prefix, msg string) {
	spacePrefix := strings.Repeat(" ", len(prefix))

	lines := strings.Split(msg, "\n")
	fmt.Printf("%s%s\n", prefix, lines[0])
	for _, line := range lines[1:] {
		fmt.Printf("%s%s\n", spacePrefix, line)
	}
}

func printFailures(t *task.Task) {
	fmt.Println()
	seenTasks := map[*task.Task]bool{}

	var printFailure func(t *task.Task, indent int)
	printFailure = func(t *task.Task, indent int) {
		strIndent := strings.Repeat("    ", indent)
		prefix := fmt.Sprintf("%s%s failed", strIndent, t.String())

		if subErr, ok := t.Error.(task.SubtasksFailure); ok {
			fmt.Printf("%s, caused by\n", prefix)
			for _, subtask := range subErr {
				printFailure(subtask, indent+1)
			}
			return
		}

		printMultilineIndented(prefix+": ", strings.TrimSuffix(t.Error.Error(), "\n"))
		if _, seen := seenTasks[t]; !seen {
			printMultilineIndented(strIndent, taskTail(t))
			seenTasks[t] = true
		} else {
			printMultilineIndented(strIndent, "<see above>")
		}
	}
	printFailure(t, 0)
}

type eventType string

const (
	eventMeta      eventType = "M"
	eventInstant   eventType = "I"
	eventStartStop eventType = "X"
)

type eventScope string

const (
	scopeGlobal eventScope = "g"
)

type eventColor string

const (
	eventColorThreadStateRunning = "thread_state_running"
)

//
// See
// https://docs.google.com/document/d/1CvAClvFfyA5R-PhYUmn5OOQtYMH4h6I0nSsKchNAySU
// for the format and allowed combinations of keys/values
//
type event struct {
	Name      string            `json:"name"`
	Category  string            `json:"cat,omitempty"`
	Type      eventType         `json:"ph"`
	ProcessID int               `json:"pid"`
	ThreadID  int               `json:"tid"`
	Args      map[string]string `json:"args,omitempty"`
	Timestamp int64             `json:"ts,omitempty"`
	Scope     eventScope        `json:"s,omitempty"`
	Duration  int64             `json:"dur,omitempty"`
	ColorName eventColor        `json:"cname,omitempty"`
}

func unixMicro(t time.Time) int64 {
	return t.UnixNano() / 1000
}

func collectEvents() []event {
	events := []event{}
	for _, task := range task.All.Tasks() {
		events = append(events, event{
			Name:     "thread_name",
			Type:     eventMeta,
			ThreadID: task.ID,
			Args: map[string]string{
				"name": task.String(),
			},
		}, event{
			Name:      "start " + task.String(),
			Type:      eventInstant,
			Scope:     scopeGlobal,
			ThreadID:  task.ID,
			Timestamp: unixMicro(task.Start()),
		}, event{
			Name:      "end " + task.String(),
			Type:      eventInstant,
			Scope:     scopeGlobal,
			ThreadID:  task.ID,
			Timestamp: unixMicro(task.End()),
		})
		for _, span := range task.Spans {
			if len(span.Subtasks) != 0 {
				continue
			}
			events = append(events, event{
				Name:      "compute",
				Category:  "compute",
				Type:      eventStartStop,
				Timestamp: unixMicro(span.Start),
				ThreadID:  task.ID,
				Duration:  span.End.Sub(span.Start).Microseconds(),
				ColorName: eventColorThreadStateRunning,
			})
		}
	}
	return events
}

func run(ctx context.Context, tasks []*task.Task, tracingFile string) (exitCode int) {
	if tracingFile != "" {
		defer func() {
			data, err := json.Marshal(collectEvents())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unexpected failure to encode JSON for tracing: %v\n", err)
				exitCode = 1
				return
			}
			if err := ioutil.WriteFile(tracingFile, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save tracing file: %v\n", err)
				exitCode = 1
				return
			}
		}()
	}

	defer func() {
		if v := recover(); v != nil {
			fmt.Printf("Unexpected error: %v\n", v)
			debug.PrintStack()
			exitCode = 1
		}
	}()

	for _, t := range tasks {
		t.Run(task.Context{Context: ctx})
		if t.Error != nil {
			printFailures(t)
			return 1
		}
	}
	return 0
}

// Main is the main function for generated Game binary
func Main(binaryName string, targets []Target, varTargets []Target, defaultTarget string, desc string, module string, usageConfig UsageConfig) {
	verbose := false
	list := false // print out a list of targets
	help := false // request target help
	var timeout time.Duration
	tracing := ""

	fs := flag.FlagSet{}
	fs.SetOutput(os.Stdout)

	// default flag set with ExitOnError and auto generated PrintDefaults should be sufficient
	fs.BoolVar(&verbose, "v", parseBool("GAMEFILE_VERBOSE"), "show verbose output when running targets")
	fs.BoolVar(&list, "l", parseBool("GAMEFILE_LIST"), "list targets for this binary")
	fs.BoolVar(&help, "h", parseBool("GAMEFILE_HELP"), "print out help for a specific target")
	fs.DurationVar(&timeout, "t", parseDuration("GAMEFILE_TIMEOUT"), "timeout in duration parsable format (e.g. 5m30s)")
	fs.StringVar(&tracing, "trace", os.Getenv("GAMEFILE_TRACE"), "trace task execution and save to the given file in Chrome trace_event format")
	fs.Usage = func() {
		fmt.Fprintf(os.Stdout, `
%s [options] [target]

Commands:
  -l    list targets in this binary
  -h    show this help

Options:
  -h    show description of a target
  -t <string>
        timeout in duration parsable format (e.g. 5m30s)
  -v    show verbose output when running targets
 `[1:], filepath.Base(os.Args[0]))
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		// flag will have printed out an error already.
		os.Exit(0)
	}
	args := fs.Args()

	if help && len(args) == 0 {
		fs.Usage()
		os.Exit(0)
	}

	log.SetFlags(0)
	if !verbose {
		log.SetOutput(ioutil.Discard)
	}
	logger := log.New(os.Stderr, "", 0)

	for _, varTarget := range varTargets {
		if _, ok := varTarget.Fn.(task.Runnable); ok {
			targets = append(targets, varTarget)
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Name < targets[j].Name
	})

	if list {
		listTargets(targets, defaultTarget, desc)
	}

	task.SetModule(module)
	task.SetReporter(consoleReporter{})

	if len(args) == 0 {
		if defaultTarget != "" {
			ignoreDefault, _ := strconv.ParseBool(os.Getenv("GAMEFILE_IGNOREDEFAULT"))
			if ignoreDefault {
				listTargets(targets, defaultTarget, desc)
			} else {
				args = []string{defaultTarget}
			}
		} else {
			listTargets(targets, defaultTarget, desc)
		}
	}

	unknown := []string{}
	for _, arg := range args {
		if findTarget(targets, arg) == nil {
			unknown = append(unknown, arg)
		}
	}
	if len(unknown) > 0 {
		logger.Printf("Unknown %s specified: %s\n", plural("target", len(unknown)),
			strings.Join(unknown, ", "))
		os.Exit(2)
	}

	if help {
		target := findTarget(targets, args[0])
		fmt.Printf("%s %s:\n\n", binaryName, target.Name)
		if target.Comment != "" {
			fmt.Print(target.Comment + "\n\n")
		}
		os.Exit(0)
	}

	if usageConfig.StateFile != "" {
		updateUsage(usageConfig, args)
	}

	ctx := context.Background()
	if timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	targetFns := []interface{}{}
	for _, targetName := range args {
		target := findTarget(targets, strings.ToLower(targetName))
		targetFns = append(targetFns, target.Fn)
	}

	tasks := task.All.Register(targetFns)

	os.Exit(run(ctx, tasks, tracing))
}
