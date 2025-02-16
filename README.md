# sadboy

呼び出し -> yobidasi -> isadiboy -> sadboy

Linter that checks whether a given function is reachable from a set of functions.

# TODO

## Handle infeasible call paths
Test7 and Test5 fail. Need to find some way to prove that paths in the generated call graph are feasible or not.

## Reflection lint is flaky

## Make golangci-lint plugin

https://golangci-lint.run/contributing/new-linters/#how-to-add-a-private-linter-to-golangci-lint

* https://github.com/gcpug/zagane/blob/master/passes/unclosetx/plugin/main.go