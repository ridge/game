package mage

// this template uses the "data"

// var only for tests
var mageMainfileTplString = `// +build ignore

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	{{- if .DefaultFunc.Name}}
	"strconv"
	{{end}}
	"strings"
	"text/tabwriter"
	{{range .Imports}}{{.UniqueName}} "{{.Path}}"
	{{end}}

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/toplevel"
)

func main() {
	mg.SetModule("{{.Module}}")

	args := toplevel.Main()

	list := func() error {
		{{with .Description}}fmt.Println(` + "`{{.}}\n`" + `)
		{{- end}}
		{{- $default := .DefaultFunc}}
		targets := map[string]string{
		{{- range .Funcs}}
			"{{lowerFirst .TargetName}}{{if and (eq .Name $default.Name) (eq .Receiver $default.Receiver)}}*{{end}}": {{printf "%q" .Synopsis}},
		{{- end}}
		{{- range .Imports}}{{$imp := .}}
			{{- range .Info.Funcs}}
			"{{lowerFirst .TargetName}}{{if and (eq .Name $default.Name) (eq .Receiver $default.Receiver)}}*{{end}}": {{printf "%q" .Synopsis}},
			{{- end}}
		{{- end}}
		}

		keys := make([]string, 0, len(targets))
		for name := range targets {
			keys = append(keys, name)
		}
		sort.Strings(keys)

		fmt.Println("Targets:")
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)
		for _, name := range keys {
			fmt.Fprintf(w, "  %v\t%v\n", name, targets[name])
		}
		err := w.Flush()
		{{- if .DefaultFunc.Name}}
			if err == nil {
				fmt.Println("\n* default target")
			}
		{{- end}}
		return err
	}

	ctx := context.Background()
	if args.Timeout != 0 {
	    var cancel context.CancelFunc
	    ctx, cancel = context.WithTimeout(ctx, args.Timeout)
	    defer cancel()
	}

	log.SetFlags(0)
	if !args.Verbose {
		log.SetOutput(ioutil.Discard)
	}
	logger := log.New(os.Stderr, "", 0)
	if args.List {
		if err := list(); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		return
	}

	targets := map[string]bool {
		{{range $alias, $funci := .Aliases}}"{{lower $alias}}": true,
		{{end}}
		{{range .Funcs}}"{{lower .TargetName}}": true,
		{{end}}
		{{range .Imports}}
			{{$imp := .}}
			{{range $alias, $funci := .Info.Aliases}}"{{if ne $imp.Alias "."}}{{lower $imp.Alias}}:{{end}}{{lower $alias}}": true,
			{{end}}
			{{range .Info.Funcs}}"{{lower .TargetName}}": true,
			{{end}}
		{{end}}
	}

	var unknown []string
	for _, arg := range args.Args {
		if !targets[strings.ToLower(arg)] {
			unknown = append(unknown, arg)
		}
	}
	if len(unknown) == 1 {
		logger.Println("Unknown target specified:", unknown[0])
		os.Exit(2)
	}
	if len(unknown) > 1 {
		logger.Println("Unknown targets specified:", strings.Join(unknown, ", "))
		os.Exit(2)
	}

	if args.Help {
		if len(args.Args) < 1 {
			logger.Println("no target specified")
			os.Exit(1)
		}
		switch strings.ToLower(args.Args[0]) {
			{{range .Funcs}}case "{{lower .TargetName}}":
				fmt.Print("{{$.BinaryName}} {{lower .TargetName}}:\n\n")
				{{if ne .Comment "" -}}
				fmt.Println({{printf "%q" .Comment}})
				fmt.Println()
				{{end}}
				var aliases []string
				{{- $name := .Name -}}
				{{- $recv := .Receiver -}}
				{{range $alias, $func := $.Aliases}}
				{{if and (eq $name $func.Name) (eq $recv $func.Receiver)}}aliases = append(aliases, "{{$alias}}"){{end -}}
				{{- end}}
				if len(aliases) > 0 {
					fmt.Printf("Aliases: %s\n\n", strings.Join(aliases, ", "))
				}
				return
			{{end}}
			default:
				logger.Printf("Unknown target: %q\n", args.Args[0])
				os.Exit(1)
		}
	}

	targetList := map[string]interface{}{}

	{{range .Funcs}}
		targetList[{{lower .TargetName | printf "%q"}}] = {{.FnName}}
	{{- end}}
	{{range .Imports}}
		{{range .Info.Funcs}}
			targetList[{{lower .TargetName | printf "%q"}}] = {{.FnName}}
		{{- end}}
	{{- end}}

	runTargets := []interface{}{}

	if len(args.Args) < 1 {
	{{- if .DefaultFunc.Name}}
		ignoreDefault, _ := strconv.ParseBool(os.Getenv("MAGEFILE_IGNOREDEFAULT"))
		if ignoreDefault {
			if err := list(); err != nil {
				logger.Println("Error:", err)
				os.Exit(1)
			}
			return
		}
		runTargets = []interface{}{ {{.DefaultFunc.FnName}} }
	{{- else}}
		if err := list(); err != nil {
			logger.Println("Error:", err)
			os.Exit(1)
		}
		return
	{{- end}}
	} else {
		for _, target := range args.Args {
			switch strings.ToLower(target) {
			{{range $alias, $func := .Aliases}}
				case "{{lower $alias}}":
					target = "{{$func.TargetName}}"
			{{- end}}
			}
			runTargets = append(runTargets, targetList[strings.ToLower(target)])
		}
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




`
