package toplevel

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/magefile/mage/mg"
)

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

// Main is the main function for generated Mage binary
func Main(binaryName string, targets []Target, defaultTarget string, desc string) {
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

	runTargets := []interface{}{}

	for _, targetName := range args {
		target := findTarget(targets, strings.ToLower(targetName))
		runTargets = append(runTargets, target.Fn)
	}

	defer func() {
		if v := recover(); v != nil {
			type code interface {
				ExitStatus() int
			}
			if c, ok := v.(code); ok {
				os.Exit(c.ExitStatus())
			}
			os.Exit(1)
		}
	}()
	mg.SerialCtxDeps(ctx, runTargets...)
}
