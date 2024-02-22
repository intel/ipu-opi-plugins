BINARY = icc
BUILD = build

all: clean $(BINARY)

$(BINARY): src/*.go
	@mkdir -p $(BUILD)
	go build -o $(BUILD)/$@ $^

clean:
	@rm -rf $(BUILD)

clean-host:
	ip addr del 192.168.11.1/24 dev ens2f0d3

clean-acc:
	ip addr del 192.168.11.2/24 dev enp0s1f0d2