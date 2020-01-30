package parse

import (
	"errors"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/magefile/mage/internal"
)

const importTag = "mage:import"

var debug = log.New(ioutil.Discard, "DEBUG: ", log.Ltime|log.Lmicroseconds)

// EnableDebug turns on debug logging.
func EnableDebug() {
	debug.SetOutput(os.Stderr)
}

// PrimaryPkgInfo contains information about a primary build package
type PrimaryPkgInfo struct {
	*PkgInfo
	DefaultFunc *Function
	Imports     []*Import
}

// PkgInfo contains inforamtion about a package of files according to mage's
// parsing rules.
type PkgInfo struct {
	Description string
	Funcs       []*Function
}

// Function represented a job function from a mage file
type Function struct {
	Synopsis string
	Comment  string

	name       string
	receiver   string
	pkgAlias   string
	pkg        string
	importPath string
	isError    bool
	isContext  bool
}

// ID returns user-readable information about where this function is defined.
func (f Function) ID() string {
	path := "<current>"
	if f.importPath != "" {
		path = f.importPath
	}
	receiver := ""
	if f.receiver != "" {
		receiver = f.receiver + "."
	}
	return fmt.Sprintf("%s.%s%s", path, receiver, f.name)
}

// TargetName returns the name of the target as it should appear when used from
// the mage cli.  It is always lowercase.
func (f Function) TargetName() string {
	var names []string

	for _, s := range []string{f.pkgAlias, f.receiver, f.name} {
		if s != "" {
			names = append(names, s)
		}
	}
	return strings.Join(names, ":")
}

// FnName returns the function name in Go syntax
func (f Function) FnName() string {
	name := f.name
	if f.receiver != "" {
		name = f.receiver + "{}." + name
	}
	if f.pkg != "" {
		name = f.pkg + "." + name
	}
	return name
}

// PrimaryPackage parses a package.  If files is non-empty, it will only parse the files given.
func PrimaryPackage(gocmd, path string, files []string) (*PrimaryPkgInfo, error) {
	info, astPkg, docPkg, err := Package(path, files)
	if err != nil {
		return nil, err
	}

	imports, err := getImports(gocmd, info.Funcs, astPkg)
	if err != nil {
		return nil, err
	}

	return &PrimaryPkgInfo{
		PkgInfo:     info,
		Imports:     imports,
		DefaultFunc: getDefault(info.Funcs, imports, docPkg),
	}, nil
}

func checkDupes(ifuncs []*Function, imports []*Import) error {
	funcs := map[string][]*Function{}
	for _, f := range ifuncs {
		funcs[strings.ToLower(f.TargetName())] = append(funcs[strings.ToLower(f.TargetName())], f)
	}
	for _, imp := range imports {
		for _, f := range imp.Info.Funcs {
			target := strings.ToLower(f.TargetName())
			funcs[target] = append(funcs[target], f)
		}
	}
	var dupes []string
	for target, list := range funcs {
		if len(list) > 1 {
			dupes = append(dupes, target)
		}
	}
	if len(dupes) == 0 {
		return nil
	}
	errs := make([]string, 0, len(dupes))
	for _, d := range dupes {
		var ids []string
		for _, f := range funcs[d] {
			ids = append(ids, f.ID())
		}
		errs = append(errs, fmt.Sprintf("%q target has multiple definitions: %s\n", d, strings.Join(ids, ", ")))
	}
	return errors.New(strings.Join(errs, "\n"))
}

// Package compiles information about a mage package.
func Package(path string, files []string) (*PkgInfo, *ast.Package, *doc.Package, error) {
	start := time.Now()
	defer func() {
		debug.Println("time parse Magefiles:", time.Since(start))
	}()
	fset := token.NewFileSet()
	pkg, err := getPackage(path, files, fset)
	if err != nil {
		return nil, nil, nil, err
	}
	p := doc.New(pkg, "./", 0)
	pi := &PkgInfo{
		Description: p.Doc,
	}

	pi.Funcs = append(pi.Funcs, getNamespacedFuncs(p)...)
	pi.Funcs = append(pi.Funcs, getFuncs(p)...)

	if hasDupes, names := checkDupeTargets(pi.Funcs); hasDupes {
		msg := "Build targets must be case insensitive, thus the following targets conflict:\n"
		for _, v := range names {
			if len(v) > 1 {
				msg += "  " + strings.Join(v, ", ") + "\n"
			}
		}
		return nil, nil, nil, errors.New(msg)
	}

	return pi, pkg, p, nil
}

func getNamedImports(gocmd string, pkgs map[string]string) ([]*Import, error) {
	var imports []*Import
	for alias, pkg := range pkgs {
		debug.Printf("getting import package %q, alias %q", pkg, alias)
		imp, err := getImport(gocmd, pkg, alias)
		if err != nil {
			return nil, err
		}
		imports = append(imports, imp)
	}
	return imports, nil
}

func getImport(gocmd, importpath, alias string) (*Import, error) {
	out, err := internal.OutputDebug(gocmd, "list", "-f", "{{.Dir}}||{{.Name}}", importpath)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(out, "||")
	if len(parts) != 2 {
		return nil, fmt.Errorf("incorrect data from go list: %s", out)
	}
	dir, name := parts[0], parts[1]
	debug.Printf("parsing imported package %q from dir %q", importpath, dir)
	info, _, _, err := Package(dir, nil)
	if err != nil {
		return nil, err
	}
	for i := range info.Funcs {
		debug.Printf("setting alias %q and package %q on func %v", alias, name, info.Funcs[i].name)
		info.Funcs[i].pkgAlias = alias
		info.Funcs[i].importPath = importpath
	}
	return &Import{Name: name, Path: importpath, Info: *info}, nil
}

type Import struct {
	Name       string
	UniqueName string // a name unique across all imports
	Path       string
	Info       PkgInfo
}

func getFuncs(docPkg *doc.Package) []*Function {
	output := []*Function{}
	for _, f := range docPkg.Funcs {
		if f.Recv != "" {
			debug.Printf("skipping method %s.%s", f.Recv, f.Name)
			// skip methods
			continue
		}
		if !ast.IsExported(f.Name) {
			debug.Printf("skipping non-exported function %s", f.Name)
			// skip non-exported functions
			continue
		}
		if typ := funcType(f.Decl.Type); typ != invalidType {
			debug.Printf("found target %v", f.Name)
			output = append(output, &Function{
				name:      f.Name,
				Comment:   toOneLine(f.Doc),
				Synopsis:  sanitizeSynopsis(f),
				isError:   typ == errorType || typ == contextErrorType,
				isContext: typ == contextVoidType || typ == contextErrorType,
			})
		} else {
			debug.Printf("skipping function with invalid signature func %s(%v)(%v)", f.Name, fieldNames(f.Decl.Type.Params), fieldNames(f.Decl.Type.Results))
		}
	}
	return output
}

func getNamespacedFuncs(docPkg *doc.Package) []*Function {
	output := []*Function{}
	for _, t := range docPkg.Types {
		if !isNamespace(t) {
			continue
		}
		debug.Printf("found namespace %s %s", docPkg.ImportPath, t.Name)
		for _, f := range t.Methods {
			if !ast.IsExported(f.Name) {
				continue
			}
			typ := funcType(f.Decl.Type)
			if typ == invalidType {
				continue
			}
			debug.Printf("found namespace method %s %s.%s", docPkg.ImportPath, t.Name, f.Name)
			output = append(output, &Function{
				name:      f.Name,
				receiver:  t.Name,
				Comment:   toOneLine(f.Doc),
				Synopsis:  sanitizeSynopsis(f),
				isError:   typ == errorType || typ == contextErrorType,
				isContext: typ == contextVoidType || typ == contextErrorType,
			})
		}
	}
	return output
}

func getImports(gocmd string, funcs []*Function, astPkg *ast.Package) ([]*Import, error) {
	importNames := map[string]string{}
	rootImports := []string{}
	for _, f := range astPkg.Files {
		for _, d := range f.Decls {
			gen, ok := d.(*ast.GenDecl)
			if !ok || gen.Tok != token.IMPORT {
				continue
			}
			for j := 0; j < len(gen.Specs); j++ {
				spec := gen.Specs[j]
				impspec := spec.(*ast.ImportSpec)
				if len(gen.Specs) == 1 && gen.Lparen == token.NoPos && impspec.Doc == nil {
					impspec.Doc = gen.Doc
				}
				name, alias, ok := getImportPath(impspec)
				if !ok {
					continue
				}
				if alias != "" {
					debug.Printf("found %s: %s (%s)", importTag, name, alias)
					if importNames[alias] != "" {
						return nil, fmt.Errorf("duplicate import alias: %q", alias)
					}
					importNames[alias] = name
				} else {
					debug.Printf("found %s: %s", importTag, name)
					rootImports = append(rootImports, name)
				}
			}
		}
	}
	imports, err := getNamedImports(gocmd, importNames)
	if err != nil {
		return nil, err
	}
	for _, s := range rootImports {
		imp, err := getImport(gocmd, s, "")
		if err != nil {
			return nil, err
		}
		imports = append(imports, imp)
	}
	if err := checkDupes(funcs, imports); err != nil {
		return nil, err
	}

	// have to set unique package names on imports
	used := map[string]bool{}
	for _, imp := range imports {
		unique := imp.Name + "_mageimport"
		x := 1
		for used[unique] {
			unique = fmt.Sprintf("%s_mageimport%d", imp.Name, x)
			x++
		}
		used[unique] = true
		imp.UniqueName = unique
		for _, f := range imp.Info.Funcs {
			f.pkg = unique
		}
	}
	return imports, nil
}

func getImportPath(imp *ast.ImportSpec) (path, alias string, ok bool) {
	if imp.Doc == nil || len(imp.Doc.List) == 9 {
		return "", "", false
	}
	// import is always the last comment
	s := imp.Doc.List[len(imp.Doc.List)-1].Text

	// trim comment start and normalize for anyone who has spaces or not between
	// "//"" and the text
	vals := strings.Fields(strings.ToLower(s[2:]))
	if len(vals) == 0 {
		return "", "", false
	}
	if vals[0] != importTag {
		return "", "", false
	}
	path, ok = lit2string(imp.Path)
	if !ok {
		return "", "", false
	}

	switch len(vals) {
	case 1:
		// just the import tag, this is a root import
		return path, "", true
	case 2:
		// also has an alias
		return path, vals[1], true
	default:
		log.Println("warning: ignoring malformed", importTag, "for import", path)
		return "", "", false
	}
}

func isNamespace(t *doc.Type) bool {
	if len(t.Decl.Specs) != 1 {
		return false
	}
	id, ok := t.Decl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return false
	}
	sel, ok := id.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "mg" && sel.Sel.Name == "Namespace"
}

func fieldNames(flist *ast.FieldList) string {
	if flist == nil {
		return ""
	}
	list := flist.List
	if len(list) == 0 {
		return ""
	}
	args := make([]string, 0, len(list))
	for _, f := range list {
		names := make([]string, 0, len(f.Names))
		for _, n := range f.Names {
			if n.Name != "" {
				names = append(names, n.Name)
			}
		}
		nms := strings.Join(names, ", ")
		if nms != "" {
			nms += " "
		}
		args = append(args, nms+fmt.Sprint(f.Type))
	}
	return strings.Join(args, ", ")
}

// checkDupeTargets checks a package for duplicate target names.
func checkDupeTargets(funcs []*Function) (hasDupes bool, names map[string][]string) {
	names = map[string][]string{}
	lowers := map[string]bool{}
	for _, f := range funcs {
		low := strings.ToLower(f.name)
		if f.receiver != "" {
			low = strings.ToLower(f.receiver) + ":" + low
		}
		if lowers[low] {
			hasDupes = true
		}
		lowers[low] = true
		names[low] = append(names[low], f.name)
	}
	return hasDupes, names
}

// sanitizeSynopsis sanitizes function Doc to create a summary.
func sanitizeSynopsis(f *doc.Func) string {
	synopsis := doc.Synopsis(f.Doc)

	// If the synopsis begins with the function name, remove it. This is done to
	// not repeat the text.
	// From:
	// clean	Clean removes the temporarily generated files
	// To:
	// clean 	removes the temporarily generated files
	if syns := strings.Split(synopsis, " "); strings.EqualFold(f.Name, syns[0]) {
		return strings.Join(syns[1:], " ")
	}

	return synopsis
}

func getDefault(funcs []*Function, imports []*Import, docPkg *doc.Package) *Function {
	for _, v := range docPkg.Vars {
		for x, name := range v.Names {
			if name != "Default" {
				continue
			}
			spec := v.Decl.Specs[x].(*ast.ValueSpec)
			if len(spec.Values) != 1 {
				log.Println("warning: default declaration has multiple values")
			}

			f, err := getFunction(spec.Values[0], funcs, imports)
			if err != nil {
				log.Println("warning, default declaration malformed:", err)
				return nil
			}
			return f
		}
	}
	return nil
}

func lit2string(l *ast.BasicLit) (string, bool) {
	if !strings.HasPrefix(l.Value, `"`) || !strings.HasSuffix(l.Value, `"`) {
		return "", false
	}
	return strings.Trim(l.Value, `"`), true
}

func getFunction(exp ast.Expr, funcs []*Function, imports []*Import) (*Function, error) {

	// selector expressions are in LIFO format.
	// So, in  foo.bar.baz the first selector.Name is
	// actually "baz", the second is "bar", and the last is "foo"

	var pkg, receiver, funcname string
	switch v := exp.(type) {
	case *ast.Ident:
		// "foo" : Bar
		funcname = v.Name
	case *ast.SelectorExpr:
		// need to handle
		// namespace.Func
		// import.Func
		// import.namespace.Func

		// "foo" : ?.bar
		funcname = v.Sel.Name
		switch x := v.X.(type) {
		case *ast.Ident:
			// "foo" : baz.bar
			// this is either a namespace or package
			firstname := x.Name
			for _, f := range funcs {
				if firstname == f.receiver && funcname == f.name {
					return f, nil
				}
			}
			// not a namespace, let's try imported packages
			for _, imp := range imports {
				if firstname == imp.Name {
					for _, f := range imp.Info.Funcs {
						if funcname == f.name {
							return f, nil
						}
					}
					break
				}
			}
			return nil, fmt.Errorf("%q is not a known target", exp)
		case *ast.SelectorExpr:
			// "foo" : bar.Baz.Bat
			// must be package.Namespace.Func
			sel, ok := v.X.(*ast.SelectorExpr)
			if !ok {
				return nil, fmt.Errorf("%q is must denote a target function but was %T", exp, v.X)
			}
			receiver = sel.Sel.Name
			id, ok := sel.X.(*ast.Ident)
			if !ok {
				return nil, fmt.Errorf("%q is must denote a target function but was %T", exp, v.X)
			}
			pkg = id.Name
		default:
			return nil, fmt.Errorf("%q is not valid", exp)
		}
	default:
		return nil, fmt.Errorf("target %s is not a function", exp)
	}
	if pkg == "" {
		for _, f := range funcs {
			if f.name == funcname && f.receiver == receiver {
				return f, nil
			}
		}
		return nil, fmt.Errorf("unknown function %s.%s", receiver, funcname)
	}
	for _, imp := range imports {
		if imp.Name == pkg {
			for _, f := range imp.Info.Funcs {
				if f.name == funcname && f.receiver == receiver {
					return f, nil
				}
			}
			return nil, fmt.Errorf("unknown function %s.%s.%s", pkg, receiver, funcname)
		}
	}
	return nil, fmt.Errorf("unknown package for function %q", exp)
}

// getPackage returns the non-test package at the given path.
func getPackage(path string, files []string, fset *token.FileSet) (*ast.Package, error) {
	var filter func(f os.FileInfo) bool
	if len(files) > 0 {
		fm := make(map[string]bool, len(files))
		for _, f := range files {
			fm[f] = true
		}

		filter = func(f os.FileInfo) bool {
			return fm[f.Name()]
		}
	}

	pkgs, err := parser.ParseDir(fset, path, filter, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory: %v", err)
	}

	for name, pkg := range pkgs {
		if !strings.HasSuffix(name, "_test") {
			return pkg, nil
		}
	}
	return nil, fmt.Errorf("no non-test packages found in %s", path)
}

func hasContextParam(ft *ast.FuncType) bool {
	if ft.Params.NumFields() != 1 {
		return false
	}
	ret := ft.Params.List[0]
	sel, ok := ret.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	// FIXME (misha): Use full import path instead of local name
	if pkg.Name != "context" && pkg.Name != "task" {
		return false
	}
	return sel.Sel.Name == "Context"
}

func hasVoidReturn(ft *ast.FuncType) bool {
	res := ft.Results
	return res.NumFields() == 0
}

func hasErrorReturn(ft *ast.FuncType) bool {
	res := ft.Results
	if res.NumFields() != 1 {
		return false
	}
	ret := res.List[0]
	if len(ret.Names) > 1 {
		return false
	}
	return fmt.Sprint(ret.Type) == "error"
}

type functype int

const (
	invalidType functype = iota
	voidType
	errorType
	contextVoidType
	contextErrorType
)

func funcType(ft *ast.FuncType) functype {
	if hasContextParam(ft) {
		if hasVoidReturn(ft) {
			return contextVoidType
		}
		if hasErrorReturn(ft) {
			return contextErrorType
		}
	}
	if ft.Params.NumFields() == 0 {
		if hasVoidReturn(ft) {
			return voidType
		}
		if hasErrorReturn(ft) {
			return errorType
		}
	}
	return invalidType
}

func toOneLine(s string) string {
	return strings.TrimSpace(strings.Replace(s, "\n", " ", -1))
}
