package parse

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	info, err := PrimaryPackage("go", "./testdata", []string{"func.go", "command.go", "repeating_synopsis.go", "subcommands.go"})
	if err != nil {
		t.Fatal(err)
	}

	expected := []Function{
		{
			name:     "ReturnsNilError",
			isError:  true,
			Comment:  "Synopsis for \"returns\" error. And some more text.",
			Synopsis: `Synopsis for "returns" error.`,
		},
		{
			name: "ReturnsVoid",
		},
		{
			name:      "TakesContextReturnsError",
			isError:   true,
			isContext: true,
		},
		{
			name:      "TakesContextReturnsVoid",
			isError:   false,
			isContext: true,
		},
		{
			name:     "RepeatingSynopsis",
			isError:  true,
			Comment:  "RepeatingSynopsis chops off the repeating function name. Some more text.",
			Synopsis: "chops off the repeating function name.",
		},
		{
			name:     "Foobar",
			receiver: "Build",
			isError:  true,
		},
		{
			name:     "Baz",
			receiver: "Build",
			isError:  false,
		},
	}

	if info.DefaultFunc == nil {
		t.Fatal("expected default func to exist, but was nil")
	}

	// DefaultisError
	if info.DefaultFunc.isError != true {
		t.Fatalf("expected DefaultIsError to be true")
	}

	// DefaultName
	if info.DefaultFunc.name != "ReturnsNilError" {
		t.Fatalf("expected DefaultName to be ReturnsNilError")
	}

	for _, fn := range expected {
		found := false
		for _, infoFn := range info.Funcs {
			if reflect.DeepEqual(fn, *infoFn) {
				found = true
				break
			} else {
				t.Logf("%#v", infoFn)
			}
		}
		if !found {
			t.Fatalf("expected:\n%#v\n\nto be in:\n%#v", fn, info.Funcs)
		}
	}

	expectedVars := []Var{
		{
			name: "VarWrongType",
		},
		{
			name: "VarNoInterface",
		},
		{
			name: "VarTarget",
		},
	}

	for _, vr := range expectedVars {
		found := false
		for _, varFn := range info.Vars {
			if reflect.DeepEqual(vr, *varFn) {
				found = true
				break
			} else {
				t.Logf("%#v", varFn)
			}
		}
		if !found {
			t.Fatalf("expected:\n%v\n\nto be in:\n%#v", vr, info.Vars)
		}
	}
}
