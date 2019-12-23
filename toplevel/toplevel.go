package toplevel

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/magefile/mage/task"
)

type consoleReporter struct {
}

func (consoleReporter) Started(t *task.Task) {
	fmt.Printf("%s STARTED %s\n", t.StringID(), t.Name())
}

func (consoleReporter) Dependencies(dependent *task.Task, dependees []*task.Task) {
	s := []string{}
	for _, d := range dependees {
		s = append(s, d.String())
	}
	fmt.Printf("%s DEPS %s -> %s\n", dependent.StringID(), dependent.Name(), strings.Join(s, ", "))
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
			cr.OutputLine(t, t.End(), task.StderrStream, line)
		}
	}
	dur := t.Duration()
	self := t.SelfDuration()
	fmt.Printf("%s %s %s time=%.02fs, self=%.02fs, subtasks=%.02fs\n",
		t.StringID(), tag, t.Name(), dur.Seconds(), self.Seconds(), (dur - self).Seconds())
}

func (consoleReporter) OutputLine(t *task.Task, time time.Time, stream task.Stream, line string) {
	tag := " "
	if stream == task.StderrStream {
		tag = "E"
	}
	fmt.Printf("%s %s | %s", t.StringID(), tag, line)
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

func printFailure(t *task.Task, indent int) {
	prefix := fmt.Sprintf("%s%s failed", strings.Repeat("    ", indent), t.String())

	if subErr, ok := t.Error.(task.SubtasksFailure); ok {
		fmt.Printf("%s, caused by\n", prefix)
		for _, subtask := range subErr {
			printFailure(subtask, indent+1)
		}
		return
	}

	printMultilineIndented(prefix+": ", strings.TrimSuffix(t.Error.Error(), "\n"))
}

// Main is the main function for generated Mage binary
func Main(binaryName string, targets []Target, defaultTarget string, desc string, module string) {
	verbose := false
	list := false // print out a list of targets
	help := false // request target help
	var timeout time.Duration

	fs := flag.FlagSet{}
	fs.SetOutput(os.Stdout)

	// default flag set with ExitOnError and auto generated PrintDefaults should be sufficient
	fs.BoolVar(&verbose, "v", parseBool("MAGEFILE_VERBOSE"), "show verbose output when running targets")
	fs.BoolVar(&list, "l", parseBool("MAGEFILE_LIST"), "list targets for this binary")
	fs.BoolVar(&help, "h", parseBool("MAGEFILE_HELP"), "print out help for a specific target")
	fs.DurationVar(&timeout, "t", parseDuration("MAGEFILE_TIMEOUT"), "timeout in duration parsable format (e.g. 5m30s)")
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

	if list {
		listTargets(targets, defaultTarget, desc)
	}

	task.SetModule(module)
	task.SetReporter(consoleReporter{})

	if len(args) == 0 {
		if defaultTarget != "" {
			ignoreDefault, _ := strconv.ParseBool(os.Getenv("MAGEFILE_IGNOREDEFAULT"))
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

	defer func() {
		if v := recover(); v != nil {
			fmt.Printf("Unexpected error: %v\n", v)
			debug.PrintStack()
			os.Exit(1)
		}
	}()

	for _, t := range tasks {
		t.Run(ctx)
		if t.Error != nil {
			printFailure(t, 0)
			os.Exit(1)
		}
	}
}
