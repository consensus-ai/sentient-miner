FROM pkienzle/opencl_docker
MAINTAINER Julian Villella <julian@objectspace.io>

ARG GOOS=linux
ARG GOARCH=amd64
ARG CC_FOR_TARGET=gcc

ARG USER=appuser

RUN groupadd -g 999 $USER && useradd -mr -u 999 -g $USER $USER

ARG GOLANG_VERSION=1.10.4

RUN apt-get update && \
  apt-get install --no-install-recommends -y -q \
    curl \
    pkg-config \
    build-essential \
    ca-certificates \
    wget \
    zip \
    bison \
    libtool \
    autoconf \
    automake \
    uuid-dev \
    checkinstall \
    git \
    mingw-w64

RUN ln -s /usr/lib/x86_64-linux-gnu/libOpenCL.so.1 /usr/lib/libOpenCL.so
COPY ./build/OpenCL.dll /usr/x86_64-w64-mingw32/lib/OpenCL.dll

RUN curl -s https://storage.googleapis.com/golang/go${GOLANG_VERSION}.linux-amd64.tar.gz | tar -v -C /usr/local -xz

ENV GOPATH=/home/$USER/go
ENV GOROOT=/usr/local/go
RUN mkdir -p $GOPATH/bin && \
    mkdir -p $GOPATH/src
ENV PATH=$PATH:$GOPATH/bin:$GOROOT/bin

RUN curl https://glide.sh/get | sh

WORKDIR $GOPATH/src/github.com/consensus-ai/sentient-miner

COPY . .
RUN chown -R $USER:$USER $GOPATH
RUN chmod g+s $GOPATH

USER $USER

RUN make clean
RUN make dependencies
# RUN make dev
RUN make release

CMD ["sh", "-c", "$GOPATH/bin/sentient-miner"]
