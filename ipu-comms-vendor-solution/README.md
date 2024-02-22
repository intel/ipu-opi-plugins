# Vendor solution for enabling the communication channel 

This project provides a vendor solution solution to detect the interface and to initialize a communication channel between the HOST and ACC. 

The Communication channel permits different compute complexes on the IPU to communicate with each other. 

This document explains how to enable communication between the ACC and the HOST. 

This project assumes that the port 3 on the HOST is mapped to port 2 on the ACC. This should be set in the ICC configuration.

## Requirements 

- Go (at least version 1.20)

## Build

```bash
make
```

*Note*: If the command above shows any errors, run `go mod tidy` to setup the required golang dependencies, then run the make command.


## Usage

Use the following commands to setup the channel:


```bash

cd build

#If the script should run on HOST 
./icc host

#If the script should run on ACC
./icc acc
```

If the IP address is set already, it can be removed using the command below:

```bash

# On HOST
ip addr del 192.168.11.1/24 dev ens2f0d3

# On ACC
ip addr del 192.168.11.2/24 dev enp0s1f0d2
```
