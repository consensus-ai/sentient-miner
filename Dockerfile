# FROM ubuntu:14.04
FROM pkienzle/opencl_docker
MAINTAINER Julian Villella <julian@objectspace.io>

ARG USER=appuser
RUN groupadd -g 999 $USER && useradd -mr -u 999 -g $USER $USER

ARG GOLANG_VERSION=1.10.4

RUN DEBIAN_FRONTEND=noninteractive \
  apt-get update && \
  apt-get install --no-install-recommends -y -q \
    curl \
    build-essential \
    ca-certificates \
    git

RUN ln -s /usr/lib/x86_64-linux-gnu/libOpenCL.so.1 /usr/lib/libOpenCL.so

RUN curl -s https://storage.googleapis.com/golang/go${GOLANG_VERSION}.linux-amd64.tar.gz | tar -v -C /usr/local -xz

ENV GOPATH=/home/$USER/go
ENV GOROOT=/usr/local/go
RUN mkdir -p $GOPATH/bin && \
    mkdir -p $GOPATH/src && \
    chown -R $USER:$USER $GOPATH
ENV PATH=$PATH:$GOPATH/bin:$GOROOT/bin

RUN curl https://glide.sh/get | sh

WORKDIR $GOPATH/src/github.com/consensus-ai/sentient-miner

COPY . .

USER $USER

# RUN make dependencies
# RUN make dev
RUN make release

CMD ["sh", "-c", "$GOPATH/bin/sentient-miner"]
