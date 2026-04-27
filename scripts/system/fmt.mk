GOFUMPT=$(shell go env GOPATH)/bin/gofumpt
GOLINES=$(shell go env GOPATH)/bin/golines
GCI=$(shell go env GOPATH)/bin/gci
WSL=$(shell go env GOPATH)/bin/wsl

fmt:
	${GOFUMPT} -l -w -extra .

lines:
	${GOLINES} -w .

gci:
	${GCI} write --skip-generated -s standard -s default -s localmodule .

wsl:
	${WSL} --fix ./...

fullfmt: fmt lines wsl gci
