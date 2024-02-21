# Package related
BINARY_NAME=ipu-dp
BUILDDIR=$(CURDIR)/build
TAG?=localhost:5000/github.com/intel/ipu-device-plugin:latest
DOCKERFILE?=$(CURDIR)/images/Dockerfile
DOCKERARGS=
ifdef HTTP_PROXY
	DOCKERARGS += --build-arg http_proxy=$(HTTP_PROXY)
endif
ifdef HTTPS_PROXY
	DOCKERARGS += --build-arg https_proxy=$(HTTPS_PROXY)
endif

.PHONY: static-check all

all: static-check build

build: 
	@echo "******     Build CNI     ******"
	go build -o ./build/$(BINARY_NAME) ./cmd/ipu-dp/
	@echo

static-check: 
	@echo "******   Verify dependencies   ******"
	go mod verify
	@echo
	@echo "******      Go Vet       ******"
	for pkg in $$(go list github.com/intel/ipu-device-plugin/...); do echo $$pkg && go vet $$pkg; done
	@echo

image:
	@docker build -t $(TAG) -f $(DOCKERFILE)  $(CURDIR) $(DOCKERARGS) --no-cache
	