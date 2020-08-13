ubuntu-deps:
	apt-get install -y software-properties-common gcc g++ python3 wget make nano seccomp libseccomp-dev
	add-apt-repository ppa:longsleep/golang-backports -y
	apt-get -y install golang-1.14-go
	cd parsley-core/script && ./install-deps

setup:
	export GOPATH=$$HOME/go
	export GOBIN=$$GOPATH/bin
	export PATH=$$PATH:/usr/lib/go-1.14/bin:$$GOBIN

core_build: setup
	cd parsley-core
	make all
	cd ..

build: core_build
	go build -o bin/main main.go
