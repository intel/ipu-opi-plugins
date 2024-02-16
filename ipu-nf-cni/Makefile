# Copyright (c) 2024 Intel Corporation.  All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License")
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

PROTOC = protoc
BINARY = nf-cni
BUILD = build
PROTOS_PATH = proto

all: clean system-check $(BUILD) protobufs $(BINARY)

build: 
	@mkdir -p $(BUILD)

$(BINARY): nf-cni/*.go 
	go build -o $(BUILD)/$@ $^

clean:
	@rm -rf $(BUILD)

deploy:	
	cp $(BUILD)/$(BINARY) /opt/cni/bin/

exec:
	kubectl exec --stdin --tty testpod1 -- /bin/bash

protobufs:
	protoc -I $(PROTOS_PATH) --go_out=$(BUILD)  --go_opt=paths=source_relative --go-grpc_out=$(BUILD) --go-grpc_opt=paths=source_relative $(PROTOS_PATH)/*.proto

.PHONY: all test clean $(BINARY)	

# The following is to test your system and ensure a smoother experience.
# They are by no means necessary to actually compile a grpc-enabled software.
PROTOC_CMD = which $(PROTOC)
PROTOC_CHECK_CMD = $(PROTOC) --version | grep -q libprotoc.3
HAS_PROTOC = $(shell $(PROTOC_CMD) > /dev/null && echo true || echo false)

SYSTEM_OK = false
ifeq ($(HAS_PROTOC),true)
SYSTEM_OK = true
endif

system-check:
ifneq ($(HAS_PROTOC),true)
	@echo " DEPENDENCY ERROR"
	@echo
	@echo "You don't have protoc 3.0.0 installed in your path."
	@echo "Please install Google protocol buffers 3.0.0 and its compiler."
	@echo "You can find it here:"
	@echo
	@echo "   https://github.com/protocolbuffers/protobuf/releases/tag/v3.0.0"
	@echo
	@echo "Here is what I get when trying to evaluate your version of protoc:"
	@echo
	-$(PROTOC) --version
	@echo
	@echo
endif
ifneq ($(SYSTEM_OK),true)
	@false
endif