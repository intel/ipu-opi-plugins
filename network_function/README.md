# Network Function Skeleton Code

This code attaches to an IPU network interface via a raw socket.
All packets received on this interface are relayed unmodified 
to a second IPU network interface. 
This provides a "transparent pipe" which can be extended to
provide true "bump in the wire" network functions.
This code supports packets up to 64K.

**To compile**
gcc -o nf nf.c

to run
./nf

To set up the test environment

Add a new namespace
ip netns add nfns
ip netns add ns1
ip netns add ns2

Create veth pairs
ip link add tap1 type veth peer name br-tap1
ip link add tap2 type veth peer name br-tap2

Push to new namespace
ip link set tap1 netns ns1
ip link set tap2 netns ns2
ip link set br-tap1 netns nfns
ip link set br-tap2 netns nfns

Assign addresses
ip netns exec ns1 ip addr add 10.1.1.4/24 dev tap1
ip netns exec ns2 ip addr add 10.1.1.5/24 dev tap2


Bring up a new interface
ip netns exec nfns ip link set br-tap1 up
ip netns exec nfns ip link set br-tap2 up
ip netns exec ns1 ip link set tap1 up
ip netns exec ns2 ip link set tap2 up

**Run code:**
ip netns exec nfns ./nf br-tap1 br-tap2

**Run test:**
ip netns exec ns1 ping -c1 10.1.1.5

Expected Behavior
packet will increment, packet output to screen, source MAC printed.
packet relayed out second interface and visible via tcpdump command
sudo ip netns exec ns2 tcpdump -vi tap2
