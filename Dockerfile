FROM debian:jessie

# allow replacing httpredir or deb mirror
ARG APT_MIRROR=deb.debian.org
RUN sed -ri "s/(httpredir|deb).debian.org/$APT_MIRROR/g" /etc/apt/sources.list

# Packaged dependencies
# NOTE: we don't need all these most likely.
# It's just very time consuming to filter these dependencies out
RUN apt-get update && apt-get install -y \
	apparmor \
	apt-utils \
	aufs-tools \
	automake \
	bash-completion \
	binutils-mingw-w64 \
	bsdmainutils \
	btrfs-tools \
	build-essential \
	clang \
	cmake \
	createrepo \
	curl \
	dpkg-sig \
	gcc-mingw-w64 \
	git \
	iptables \
	jq \
	less \
	libapparmor-dev \
	libcap-dev \
	libltdl-dev \
	libnl-3-dev \
	libprotobuf-c0-dev \
	libprotobuf-dev \
	libsystemd-journal-dev \
	libtool \
	mercurial \
	net-tools \
	pkg-config \
	protobuf-compiler \
	protobuf-c-compiler \
	python-dev \
	python-mock \
	python-pip \
	python-websocket \
	tar \
	vim \
	vim-common \
	xfsprogs \
	zip \
	--no-install-recommends \
	&& pip install awscli==1.10.15

# Get lvm2 source for compiling statically
ENV LVM2_VERSION 2.02.103
RUN mkdir -p /usr/local/lvm2 \
	&& curl -fsSL "https://mirrors.kernel.org/sourceware/lvm2/LVM2.${LVM2_VERSION}.tgz" \
		| tar -xzC /usr/local/lvm2 --strip-components=1
# See https://git.fedorahosted.org/cgit/lvm2.git/refs/tags for release tags

# Compile and install lvm2
RUN cd /usr/local/lvm2 \
	&& ./configure \
		--build="$(gcc -print-multiarch)" \
		--enable-static_link \
	&& make device-mapper \
	&& make install_device-mapper
# See https://git.fedorahosted.org/cgit/lvm2.git/tree/INSTALL

# Configure the container for OSX cross compilation
ENV OSX_SDK MacOSX10.11.sdk
ENV OSX_CROSS_COMMIT a9317c18a3a457ca0a657f08cc4d0d43c6cf8953
RUN set -x \
	&& export OSXCROSS_PATH="/osxcross" \
	&& git clone https://github.com/tpoechtrager/osxcross.git $OSXCROSS_PATH \
	&& ( cd $OSXCROSS_PATH && git checkout -q $OSX_CROSS_COMMIT) \
	&& curl -sSL https://s3.dockerproject.org/darwin/v2/${OSX_SDK}.tar.xz -o "${OSXCROSS_PATH}/tarballs/${OSX_SDK}.tar.xz" \
	&& UNATTENDED=yes OSX_VERSION_MIN=10.6 ${OSXCROSS_PATH}/build.sh
ENV PATH /osxcross/target/bin:$PATH

# Install Go
ENV GO_VERSION 1.8.3
RUN curl -fsSL "https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
	| tar -xzC /usr/local

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

# Compile Go for cross compilation
ENV DOCKER_CROSSPLATFORMS \
	linux/386 linux/arm \
	darwin/amd64 \
	freebsd/amd64 freebsd/386 freebsd/arm \
	windows/amd64 windows/386 \
	solaris/amd64

# Dependency for golint
ENV GO_TOOLS_COMMIT 823804e1ae08dbb14eb807afc7db9993bc9e3cc3
RUN git clone https://github.com/golang/tools.git /go/src/golang.org/x/tools \
	&& (cd /go/src/golang.org/x/tools && git checkout -q $GO_TOOLS_COMMIT)

# Grab Go's lint tool
ENV GO_LINT_COMMIT 32a87160691b3c96046c0c678fe57c5bef761456
RUN git clone https://github.com/golang/lint.git /go/src/github.com/golang/lint \
	&& (cd /go/src/github.com/golang/lint && git checkout -q $GO_LINT_COMMIT) \
	&& go install -v github.com/golang/lint/golint

ADD . /build-root
WORKDIR /build-root
CMD make
