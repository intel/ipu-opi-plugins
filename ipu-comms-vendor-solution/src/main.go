package main

import (
	"os"
	"strings"

	"github.com/vishvananda/netlink"
)

const (
	CONFIG_FILE_PATH = "../config/config.yaml"

	HOST_VPORT_ID = "03"
	ACC_VPORT_ID  = "02"

	DEVICE = "0x1452"
	VENDOR = "0x8086"
)

func isPF(pfName string) bool {
	device, _ := getDevice(pfName)
	vendor, _ := getVendor(pfName)
	return strings.TrimSpace(string(device)) == DEVICE && strings.TrimSpace(string(vendor)) == VENDOR
}

func setIP(pf netlink.Link, ip string) {
	// Get the list of current IP addresses
	list, _ := netlink.AddrList(pf, netlink.FAMILY_V4)
	if len(list) == 0 {

		// Set the IP address on PF
		addr, _ := netlink.ParseAddr(ip)
		netlink.AddrAdd(pf, addr)
	}
}

func setHost() {
	// Identify the PF allocated for HOST-ACC comm
	linkList, _ := netlink.LinkList()
	var pf netlink.Link
	for i := 0; i < len(linkList); i++ {
		if isPF(linkList[i].Attrs().Name) {
			mac := linkList[i].Attrs().HardwareAddr.String()
			octets := strings.Split(mac, ":")

			// Check the 4th octet which is used to identify the PF
			if octets[3] == HOST_VPORT_ID {
				pf = linkList[i]
				break
			}
		}
	}

	if pf == nil {
		exitWithError("Unable to identify any PFs. Check that the idpf driver is available", 1)
	}

	// Get the IP address to assign to the PF
	var cfg Config
	readConfig(&cfg, CONFIG_FILE_PATH)

	setIP(pf, cfg.Host.Ip)
}

func setACC() {
	// Identify the PF allocated for HOST-ACC comm
	linkList, _ := netlink.LinkList()
	var pf netlink.Link
	for i := 0; i < len(linkList); i++ {
		if isPF(linkList[i].Attrs().Name) {
			mac := linkList[i].Attrs().HardwareAddr.String()
			octets := strings.Split(mac, ":")

			// Check the 4th octet which is used to identify the PF
			if octets[3] == ACC_VPORT_ID {

				// On ACC, the 4th octet in the base mac address may already be set to ACC_VPORT_ID
				// The base mac address is set to be used by the PF mapped to the IMC, and has an IP address.
				// Two distinguish between the two, we select the one which doesn't have an IP set already.
				if list, _ := netlink.AddrList(linkList[i], netlink.FAMILY_V4); len(list) == 0 {
					pf = linkList[i]
					break
				}
			}
		}
	}

	if pf == nil {
		exitWithError("Unable to identify any PFs. Check that the idpf driver is available", 1)
	}

	// Get the IP address to assign to the PF
	var cfg Config
	readConfig(&cfg, CONFIG_FILE_PATH)

	setIP(pf, cfg.ACC.Ip)
}

func main() {

	mode := "host"

	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	if mode == "host" {
		setHost()
	} else if mode == "acc" {
		setACC()
	} else {
		exitWithError("The mode provided is not available. Select either \"host\" or \"acc\".", 1)
	}
}
