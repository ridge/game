package mage

// this template uses the "data"

// var only for tests
var mageMainfileTplString = `// +build ignore

package main

import (
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

	vtlt := []toplevel.Target{
{{- range .Vars}}
		toplevel.Target{Name: {{lowerFirst .TargetName | printf "%q"}},
			Fn: {{.VarName}},
			Synopsis: {{printf "%q" .Synopsis}},
			Comment: {{printf "%q" .Comment}}},
{{- end}}
{{- range .Imports}}
{{- range .Info.Vars}}
		toplevel.Target{Name: {{lowerFirst .TargetName | printf "%q"}},
			Fn: {{.VarName}},
			Synopsis: {{printf "%q" .Synopsis}},
			Comment: {{printf "%q" .Comment}}},
{{- end}}
{{- end}}
	}

	toplevel.Main({{printf "%q" $.BinaryName}}, tlt, vtlt,
		{{lowerFirst .DefaultFunc.TargetName | printf "%q"}},
		{{printf "%q" .Description}},
		{{.Module | printf "%q"}})
}




`
