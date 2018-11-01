# These variables get inserted into ./build/commit.go
BUILD_TIME=$(shell date)
GIT_REVISION=$(shell git rev-parse --short HEAD)
GIT_DIRTY=$(shell git diff-index --quiet HEAD -- || echo "✗-")

# sets values of version string variables
ldflags= -X "github.com/consensus-ai/sentient-miner/build.GitRevision=${GIT_DIRTY}${GIT_REVISION}" \
-X "github.com/consensus-ai/sentient-miner/build.BuildTime=${BUILD_TIME}"

# all will build and install release binaries
all: release

# dependencies installs all of the dependencies that are required for building
# Sen.
dependencies:
	glide install

dependencies-update:
	glide cc && glide up

# pkgs changes which packages the makefile calls operate on. run changes which
# tests are run during testing.
run = .
pkgs = ./mining \
       ./clients \
       ./clients/stratum \
       ./algorithms/sentient \
       .

# fmt calls go fmt on all packages.
fmt:
	gofmt -s -l -w $(pkgs)

lint:
	golint -min_confidence=1.0 -set_exit_status $(pkgs)

# spellcheck checks for misspelled words in comments or strings.
spellcheck:
	misspell -error .

# dev builds and installs developer binaries.
dev:
	go install -tags='dev debug netgo' -ldflags='$(ldflags)' $(pkgs)

# release builds and installs release binaries.
release:
	go install -tags='netgo' -a -ldflags='-s -w $(ldflags)' $(pkgs)

# clean removes all directories that get automatically created during
# development.
clean:
	rm -rf release && go clean -testcache && go clean -i github.com/consensus-ai/sentient-miner

test:
	go test -v -short -tags='debug testing netgo' -timeout=15s $(pkgs) -run=$(run)
test-v:
	go test -race -v -short -tags='debug testing netgo' -timeout=15s $(pkgs) -run=$(run)
test-long: clean fmt vet lint
	go test -v -race -tags='testing debug netgo' -timeout=500s $(pkgs) -run=$(run)
test-vlong: clean fmt vet lint
	go test -v -race -tags='testing debug vlong netgo' -timeout=5000s $(pkgs) -run=$(run)
