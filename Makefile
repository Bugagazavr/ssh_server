all: clean deps build test

deps:
	go get github.com/karalabe/xgo
	go get ./...
	bundle install -j8

build:
	xgo --buildmode=c-shared --targets="darwin/amd64,darwin/386,linux/amd64,linux/386" -dest=ext -out=ssh_server .

clean:
	rm -rf ./ext/*

test:
	bundle exec rspec
