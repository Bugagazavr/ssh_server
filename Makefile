all: clean deps build

deps:
	go get ./...

build:
	xgo --buildmode=c-shared --targets="darwin/amd64,darwin/386,linux/amd64,linux/386" -dest=ext -out=ssh_server .

clean:
	rm -rf ./ext/*
