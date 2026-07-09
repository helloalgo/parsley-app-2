ubuntu-deps:
	apt-get install -y software-properties-common gcc g++ python3 wget make nano seccomp libseccomp-dev strace
	add-apt-repository ppa:longsleep/golang-backports -y
	add-apt-repository ppa:ubuntu-toolchain-r/test -y
	apt-get update
	apt-get install -y gcc-11 g++-11
	update-alternatives --install /usr/bin/gcc gcc /usr/bin/gcc-11 100
	update-alternatives --install /usr/bin/g++ g++ /usr/bin/g++-11 100
	apt-get -y install golang-1.14-go

core_build:
	cd parsley-core && make all

build: core_build
	go build -o bin/main main.go

docker:
	docker build -f build/Dockerfile -t jungnoh/parsley-app:latest .
