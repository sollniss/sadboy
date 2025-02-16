package analyzer_test

import (
	"testing"

	"github.com/sollniss/sadboy/analyzer"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestCallers(t *testing.T) {
	testdata := analysistest.TestData()
	defer analyzer.SetOpts(func(o *analyzer.Opts, caller *analyzer.CallerOpts, callee *analyzer.CalleeOpts) {
		caller.Params = []string{"callers/caller.Param"}
		caller.Results = []string{"callers/caller.Result"}

		callee.Name = "Callee"
	})()
	analysistest.Run(t, testdata, analyzer.Analyzer, "callers/...")
}
