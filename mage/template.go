package mage

// this template uses the "data"

// var only for tests
var mageMainfileTplString = `// +build ignore

package main

import (
	"sort"
	{{range .Imports}}{{.UniqueName}} "{{.Path}}"
	{{end}}

	"github.com/magefile/mage/toplevel"
)

func main() {
	tlt := []toplevel.Target{
{{- range .Funcs}}
		toplevel.Target{Name: {{lowerFirst .TargetName | printf "%q"}},
			Fn: {{.FnName}},
			Synopsis: {{printf "%q" .Synopsis}},
			Comment: {{printf "%q" .Comment}}},
{{- end}}
{{- range .Imports}}
{{- range .Info.Funcs}}
		toplevel.Target{Name: {{lowerFirst .TargetName | printf "%q"}},
			Fn: {{.FnName}},
			Synopsis: {{printf "%q" .Synopsis}},
			Comment: {{printf "%q" .Comment}}},
{{- end}}
{{- end}}
	}
	sort.Slice(tlt, func(i, j int) bool {
		return tlt[i].Name<tlt[j].Name
	})

	toplevel.Main({{printf "%q" $.BinaryName}}, tlt,
		{{lowerFirst .DefaultFunc.TargetName | printf "%q"}},
		{{printf "%q" .Description}},
		{{.Module | printf "%q"}})
}




`
