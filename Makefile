ubuntu-deps:
	apt-get install -y software-properties-common gcc g++ python3 wget make nano seccomp libseccomp-dev strace
	add-apt-repository ppa:longsleep/golang-backports -y
	apt-get -y install golang-1.14-go

core_build:
	cd parsley-core && make all

build: core_build
	go build -o bin/main main.go

docker:
	docker build -f build/Dockerfile -t jungnoh/parsley-app:latest .