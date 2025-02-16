package analyzer

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/vta"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	_ "unsafe" // for linkname hack
)

func init() {
	Analyzer.Flags.Func("skip.file", "skip all files with specified suffixes", setSlice(&opts.SkipFileSuffixes))

	Analyzer.Flags.StringVar(&calleeOpts.Name, "callee.name", "", "callee function name")
	Analyzer.Flags.Func("callee.params", "callee function params (comma separated, in order)", setSlice(&calleeOpts.Params))
	Analyzer.Flags.Func("callee.results", "callee function results (comma separated, in order)", setSlice(&calleeOpts.Results))
	Analyzer.Flags.Func("callee.pkg", "callee function package prefix", setSlice(&calleeOpts.PkgPrefixes))

	Analyzer.Flags.Func("caller.names", "caller function names (comma separated)", setMap(&callerOpts.Names))
	Analyzer.Flags.Func("caller.params", "caller function params (comma separated, in order)", setSlice(&callerOpts.Params))
	Analyzer.Flags.Func("caller.results", "caller function results (comma separated, in order)", setSlice(&callerOpts.Results))
	Analyzer.Flags.Func("caller.pkg", "caller function package prefix", setSlice(&callerOpts.PkgPrefixes))
}

func setSlice(o *[]string) func(string) error {
	return func(s string) error {
		if s != "" {
			*o = strings.Split(s, ",")
		}
		return nil
	}
}

func setMap(o *map[string]struct{}) func(string) error {
	return func(s string) error {
		if s != "" {
			f := strings.Split(s, ",")
			*o = make(map[string]struct{}, len(f))
			for _, n := range f {
				(*o)[n] = struct{}{}
			}
		}
		return nil
	}
}

var (
	opts       Opts
	callerOpts CallerOpts
	calleeOpts CalleeOpts
)

type Opts struct {
	// Skip callers and callees in all files with specified suffixes.
	SkipFileSuffixes []string
}

type CallerOpts struct {
	// Function names to search for.
	Names map[string]struct{}

	// Callers in a package not containing a prefix are skipped.
	PkgPrefixes []string

	// Types of the function's parameters.
	Params []string

	// Types of the function's results (return types).
	Results []string
}

type CalleeOpts struct {
	// Function name to search for.
	Name string

	// Callees in a package not containing a prefix are skipped.
	PkgPrefixes []string

	// Types of the function's parameters.
	Params []string

	// Types of the function's results (return types).
	Results []string
}

type typesFact struct {
	typesInfo *types.Info
}

func (*typesFact) AFact() {}

func (*typesFact) String() string {
	return "types"
}

var Analyzer = &analysis.Analyzer{
	Name: "sadboy",
	Doc:  "checks if there exists a call path between caller and callee",
	Run:  run,
	Requires: []*analysis.Analyzer{
		AnalyzerHasCaller,
	},
	FactTypes: []analysis.Fact{
		&typesFact{},
	},
}

func run(pass *analysis.Pass) (interface{}, error) {

	preScanRes, ok := pass.ResultOf[AnalyzerHasCaller].(*preScanResult)
	if !ok {
		panic("no pre scan result")
	}

	// Export type info of the current package.
	fact := typesFact{pass.TypesInfo}
	pass.ExportPackageFact(&fact)

	if !preScanRes.hasCaller {
		return nil, nil
	}

	// Build program for current package first.

	prog := ssa.NewProgram(pass.Fset, ssa.InstantiateGenerics)

	seen := make(map[*types.Package]struct{})

	// Add imported packages to the program.
	// This is similar to what buildssa.Analyzer does, but we also build imported packages.
	var addImports func(pp []*types.Package, depth int)
	addImports = func(pp []*types.Package, depth int) {
		//if depth > 2 {
		//	return
		//}

		for _, p := range pp {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}

			tfact := &typesFact{}
			// To build the imported package, we need it's type info.
			if ok := pass.ImportPackageFact(p, tfact); !ok {
				// No data. Create as dummy.
				//pass.Reportf(0, "%s :: %s is dummy", pass.Pkg.Path(), p.Path())
				pkg := prog.CreatePackage(p, nil, nil, true)
				pkg.Build()
				continue
			}

			// Get package files from type info.
			files := make([]*ast.File, len(tfact.typesInfo.FileVersions))
			i := 0
			for f := range tfact.typesInfo.FileVersions {
				files[i] = f
				i++
			}
			// Add package to program.
			//pass.Reportf(0, "%s :: importing %s", pass.Pkg.Path(), p.Path())
			pkg := prog.CreatePackage(p, files, tfact.typesInfo, true)

			// Add imports of the imported package.
			addImports(p.Imports(), depth+1)

			pkg.Build()
		}
	}
	addImports(pass.Pkg.Imports(), 0)
	//return nil, nil

	// Create and build the primary package (copied from buildssa.Analyzer).
	prog.CreatePackage(pass.Pkg, pass.Files, pass.TypesInfo, false)
	prog.Build()

	progFns := ssautil.AllFunctions(prog)

	callerFns := make(map[*ssa.Function]struct{})
	for fn := range progFns {
		// Root node.
		if fn == nil {
			continue
		}

		if fn.Pkg == nil || fn.Pkg.Pkg != pass.Pkg {
			continue
		}

		// Skip synthetic functions.
		// These could match signatures of the target callers,
		// and therefore cause early termination of the search.
		if isSynthetic(fn) {
			continue
		}

		// Check if file should be skipped.
		// We don't actually skip here, since we need to keep track of all caller functions (even the ones skipped).
		// If we would skip here, we might end up in weird places in the call graph when following the caller forever.
		// This is probably not required for all code bases.
		file := prog.Fset.File(fn.Pos())
		var skip bool
		if file != nil {
			fileName := file.Name()
			for i := len(opts.SkipFileSuffixes) - 1; i >= 0; i-- {
				if strings.HasSuffix(fileName, opts.SkipFileSuffixes[i]) {
					skip = true
					break
				}
			}
		}

		// Check if function signature matches caller.
		if !chkSig(fn.Signature, callerOpts.Params, callerOpts.Results) {
			continue
		}

		// If name is specified, all callers must match.
		if len(callerOpts.Names) > 0 {
			if _, ok := callerOpts.Names[fn.Name()]; !ok {
				continue
			}
		}

		// Skip if file should be skipped.
		if skip {
			continue
		}

		// Record target caller.
		callerFns[fn] = struct{}{}
	}

	//pass.Reportf(1, "%s :callers %d", pass.Pkg.Path(), len(callerFns))
	if len(callerFns) == 0 {
		return nil, nil
	}

	// Build call graph.
	// No need to do CHA first.
	cg := vta.CallGraph(progFns, nil)

	//pass.Reportf(1, "call graph for %s:\n%s", pass.Pkg.Path(), cgToString(cg))

	// Deleting synthetic nodes would remove calls to functions outside of the package.
	//cg.DeleteSyntheticNodes()

	foundCallers := make(map[*ssa.Function]struct{}, len(callerFns))
	for caller := range callerFns {
		path := PathSearch(pass, cg.Nodes[caller], func(n *callgraph.Node) bool {
			// Check if function name matches callee.
			if n.Func.Name() != calleeOpts.Name {
				return false
			}
			// Check if function signature matches callee.
			if !chkSig(n.Func.Signature, calleeOpts.Params, calleeOpts.Results) {
				return false
			}

			//pass.Reportf(1, "found callee: %s -> %s", caller.RelString(nil), n.Func.RelString(nil))
			return true
		})
		if path != nil {
			foundCallers[caller] = struct{}{}
		}
	}

	for fn := range callerFns {
		if _, ok := foundCallers[fn]; !ok {
			pass.Reportf(fn.Pos(), "%s does not call callee function", fn.Name())
		}
	}

	return nil, nil
}

// PathSearch finds an arbitrary path starting at node start and
// ending at some node for which isEnd() returns true.  On success,
// PathSearch returns the path as an ordered list of edges; on
// failure, it returns nil.
//
// copied and modified from [callgraph.PathSearch].
func PathSearch(pass *analysis.Pass, start *callgraph.Node, isEnd func(*callgraph.Node) bool) []*callgraph.Edge {
	stack := make([]*callgraph.Edge, 0, 32)
	seen := make(map[*callgraph.Node]struct{})

	// Check if caller has a function param,
	// and see if it's actually called at this site.
	//
	// This is required because the call graph contains all possible
	// calls, and we need to find the correct edge for this call.
	//
	// Example:
	// func A(a func(){}) { a() }
	//
	// A(B)
	//
	// `a` will have two outgoing edges to `B` and to `C`,
	// but we don't know which one it is by just looking at the call site `a()`.
	// We also need to check the incoming edge `inc` to this call site.
	var isFakeCall func(inc *callgraph.Edge, e *callgraph.Edge) bool
	isFakeCall = func(inc *callgraph.Edge, e *callgraph.Edge) bool {
		// e.Caller.Func would be `A` in the example above.
		callerParams := e.Caller.Func.Params
		// Caller has no params, nothing to do.
		if len(callerParams) == 0 {
			//pass.Reportf(e.Site.Pos(), "caller has no params")
			return false
		}

		common := inc.Site.Common()
		var hasFunc bool

		var paramIdx int
		var argIdx int
		if common.IsInvoke() {
			paramIdx = 1
		}
		for ; paramIdx < len(callerParams); paramIdx, argIdx = paramIdx+1, argIdx+1 {
			param := callerParams[paramIdx]
			// Check if param is a function.
			if _, ok := param.Type().Underlying().(*types.Signature); ok {
				hasFunc = true

				//if len(callerParams) != len(common.Args) {
				//	panic(fmt.Sprintf("\n"+
				//		"callerCallSite: %s\n"+
				//		"caller1: %s\n"+
				//		"caller2: %s\n"+
				//		"callerParams: %s\n"+
				//		"param[%d]: %s\n"+
				//		"commonArgs: %s\n"+
				//		"isInvoke: %t\n",
				//		inc.Site, inc.Callee.Func, e.Caller.Func, callerParams, i, param, common.Args, common.IsInvoke(),
				//	))
				//}

				arg := common.Args[argIdx]

				var incCallParam *ssa.Function
				switch arg := arg.(type) {
				case *ssa.Function:
					incCallParam = arg
				case *ssa.MakeClosure:
					// TODO: make test case for this and confirm this is correct
					incCallParam = arg.Fn.(*ssa.Function)
				case *ssa.Parameter:
					// Arg is passed down as a param.
					//pass.Reportf(e.Site.Pos(), "param [%s] is arg #%d [%s]", param.Name(), argIdx, arg.Name())
					return false
				default:
					//pass.Reportf(e.Site.Pos(), "unknown arg type: %T", arg)
				}

				// Check if incomming call param and arg are the same function.
				// (a == B in the example above)
				if incCallParam == e.Callee.Func {
					//pass.Reportf(e.Site.Pos(), "param [%s] is called", param.Name())
					return false
				}

				// Check if param is passed down as an arg.
				for _, cArg := range e.Site.Common().Args {
					if param.Name() == cArg.Name() {
						//pass.Reportf(
						//	e.Site.Pos(),
						//	"param [%s] is arg with same name [%s]",
						//	param.Name(), cArg.Name(),
						//)
						return false
					}

					// case *ssa.Parameter should be covered by the name match above.
					var outCallParam *ssa.Function
					switch arg := cArg.(type) {
					case *ssa.Function:
						outCallParam = arg
					case *ssa.MakeClosure:
						outCallParam = arg.Fn.(*ssa.Function)
					}

					// A(func(){})
					if incCallParam == outCallParam {
						//pass.Reportf(e.Site.Pos(), "param [%s] is arg with diff name [%s]", param.Name(), cArg.Name())
						return false
					}
				}

				// param is neither called nor passes as an arg.
				//pass.Reportf(e.Site.Pos(), "param [%s] neither called nor passed as an arg", param.Name())
				return true
			}
		}

		//if !hasFunc {
		//	pass.Reportf(e.Site.Pos(), "pass has func")
		//}

		//pass.Reportf(e.Site.Pos(), "call has no func param %t", hasFunc)
		return hasFunc
	}
	var search func(n *callgraph.Node) []*callgraph.Edge
	search = func(n *callgraph.Node) []*callgraph.Edge {
		if _, ok := seen[n]; !ok {
			seen[n] = struct{}{}
			if isEnd(n) {
				return stack
			}
			for _, e := range n.Out {
				// TODO: check len(n.Out) and only call isFakeCall if len(n.Out) > 1 ??
				if len(stack) > 0 && isFakeCall(stack[len(stack)-1], e) {
					//pass.Reportf(e.Site.Pos(),
					//	"fake call: %s -> %s",
					//	stack[len(stack)-1].Site,
					//	e.Site,
					//)
					continue
				}
				stack = append(stack, e) // push
				if found := search(e.Callee); found != nil {
					return found
				}
				stack = stack[:len(stack)-1] // pop
			}
		}
		return nil
	}
	return search(start)
}

func debug(condition bool, reportf func()) {
	if condition {
		reportf()
	}
}

func cgToString(cg *callgraph.Graph) string {
	var str string
	for _, n := range cg.Nodes {
		str += fmt.Sprintf("%s\n", n.Func.RelString(nil))
		for _, e := range n.Out {
			str += fmt.Sprintf("\t-> %s\n", e.Callee.Func.RelString(nil))
		}
		for _, e := range n.In {
			str += fmt.Sprintf("\t<- %s\n", e.Caller.Func.RelString(nil))
		}
	}
	return str
}

// chkPkg returns true if pkg matches a prefix in chk
// or if either are nil.
func checkPkg(pkg *types.Package, chk []string) bool {
	if pkg == nil || chk == nil {
		return true
	}
	path := pkg.Path()
	for i := len(chk) - 1; i >= 0; i-- {
		if strings.HasPrefix(path, chk[i]) {
			return true
		}
	}
	return false
}

func chkSig(sig *types.Signature, params, results []string) bool {
	// nil means unset, len() == 0 means no params/results.

	if params != nil {
		fnParams := sig.Params()
		if fnParams.Len() != len(params) {
			return false
		}

		// Since many functions start with [context.Context] we check in reverse order
		// to find missmatches faster.
		for i := len(params) - 1; i >= 0; i-- {
			if fnParams.At(i).Type().String() != params[i] {
				return false
			}
		}
	}

	if results != nil {
		fnResults := sig.Results()
		if fnResults.Len() != len(results) {
			return false
		}

		// Almost all functions return an [error] as the last return value,
		// so we match in normal order here.
		for i := 0; i < len(results); i++ {
			if fnResults.At(i).Type().String() != results[i] {
				return false
			}
		}
	}

	return true
}

// isSynthetic returns true if the function has no representation in the source code.
// Copied from DeleteSyntheticNodes.
func isSynthetic(fn *ssa.Function) bool {
	if isInit(fn) || fn.Syntax() != nil {
		return false
	}
	return true
}

func isInit(fn *ssa.Function) bool {
	return fn.Pkg != nil && fn.Pkg.Func("init") == fn
}

var AnalyzerHasCaller = &analysis.Analyzer{
	Name: "sadboy_hascaller",
	Doc:  "checks if the package contains any callers",
	Run:  runHasCallers,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
	ResultType: reflect.TypeOf(new(preScanResult)),
}

type preScanResult struct {
	hasCaller bool
}

func runHasCallers(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil)}

	var hasCaller bool
	inspector.Preorder(nodeFilter, func(n ast.Node) {
		if !hasCaller {
			fn := n.(*ast.FuncDecl)
			sig := pass.TypesInfo.TypeOf(fn.Name).(*types.Signature)

			nameMatch := true
			if len(callerOpts.Names) > 0 {
				_, nameMatch = callerOpts.Names[fn.Name.Name]
			}
			if nameMatch &&
				chkSig(sig, callerOpts.Params, callerOpts.Results) &&
				checkPkg(pass.Pkg, callerOpts.PkgPrefixes) {
				hasCaller = true
			}
		}
	})

	return &preScanResult{
		hasCaller: hasCaller,
	}, nil
}
