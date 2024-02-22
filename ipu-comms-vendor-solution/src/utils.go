package main

import (
	"fmt"
	"os"
)

func exitWithError(err string, exitCode int) {
	fmt.Fprintf(os.Stderr, "Error : %s\n", err)
	os.Exit(exitCode)
}

func getDevice(ifName string) ([]byte, error) {
	return os.ReadFile(fmt.Sprintf("/sys/class/net/%s/device/device", ifName))
}

func getVendor(ifName string) ([]byte, error) {
	return os.ReadFile(fmt.Sprintf("/sys/class/net/%s/device/vendor", ifName))
}
