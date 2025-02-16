callers:
	go build ./cmd/sadboy
	cd analyzer/testdata/src/callers && ../../../../sadboy -callee.name=Callee -caller.params=callers/caller.Param -caller.results=callers/caller.Result ./...

c:
	go build ./cmd/sadboy
	cd analyzer/testdata/src/pkgtest && ../../../../sadboy -callee.name=A -caller.results=pkgtest/pkg3.Return -test=false ./...

nilaway:
	go install go.uber.org/nilaway/cmd/nilaway@latest
	cd analyzer/testdata/src/nilaway && nilaway -experimental-anonymous-function=true ./...

wally:
	go install github.com/hex0punk/wally@latest
	cd analyzer/testdata/src/wally && wally map search -p ./... --func B --pkg wally/b --ssa -vvv