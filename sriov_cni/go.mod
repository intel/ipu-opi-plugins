module github.com/k8snetworkplumbingwg/sriov-cni

go 1.20

require (
	github.com/containernetworking/cni v1.1.2
	github.com/containernetworking/plugins v1.2.0
	github.com/onsi/ginkgo/v2 v2.9.2
	github.com/onsi/gomega v1.27.5
	github.com/opiproject/opi-api v0.0.0-20230831141553-56e46f754823
	github.com/stretchr/testify v1.6.1
	github.com/vishvananda/netlink v1.2.1-beta.2
	golang.org/x/net v0.17.0
	google.golang.org/grpc v1.57.0
)

require (
	github.com/coreos/go-iptables v0.6.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/pprof v0.0.0-20230323073829-e72429f035bd // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/safchain/ethtool v0.2.0 // indirect
	github.com/stretchr/objx v0.1.0 // indirect
	github.com/vishvananda/netns v0.0.2 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20230803162519-f966b187b2e5 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230803162519-f966b187b2e5 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/onsi/ginkgo/v2 => github.com/onsi/ginkgo/v2 v2.9.2
	github.com/onsi/gomega => github.com/onsi/gomega v1.27.5
)
