// Copyright (c) 2023 Intel Corporation.  All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License")
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/ipuplugin"
	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/p4rtclient"
	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/types"
	"github.com/ipdk-io/k8s-infra-offload/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	configFilePath      = "/etc/ipu/"
	cliName             = "ipuplugin"
	defaultServingAddr  = "/var/run/vendor-plugin.sock"
	defaultServingPort  = 50152
	defaultServingProto = "unix"
	tenantBridgeName    = "br-tenant"
	defaultLogDir       = "/var/log/ipuplugin"
	defaultBridge       = "linux"
	defaultP4rtBin      = "/opt/p4/p4-cp-nws/bin/p4rt-ctl"
	defaultOvsCliDir    = "/opt/p4/p4-cp-nws/bin"
	defaultPortMuxVsi   = 0x0a
	defaultP4BridgeName = "br0"
	defaultDaemonHostIp = "192.168.1.1"
	defaultDaemonIpuIp  = "192.168.1.2"
	defaultDaemonPort   = 50151
)

var (
	config struct {
		cfgFile       string
		port          int
		servingAddr   string
		servingProto  string
		bridge        string
		interfaceName string
		logDir        string
		ovsCliDir     string
		bridgeType    string
		p4rtbin       string
		portMuxVsi    int
		verbosity     string
		mode          string
		daemonHostIp  string
		daemonIpuIp   string
		daemonPort    int
	}

	rootCmd = &cobra.Command{
		Use:   cliName,
		Short: "IPU plugin is a daemon that exposes a CNI gRPC backend for SR-IOV networking offload to Intel MEV.",
		Long: `IPU plugin is a daemon that exposes a CNI gRPC backend for SR-IOV networking offload to Intel MEV.
		`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateConfigs()
		},
		Run: func(_ *cobra.Command, _ []string) {
			if err := logInit(viper.GetString("logDir"), viper.GetString("verbosity")); err != nil {
				exitWithError(err, 3)
			}

			servingAddr := viper.GetString("servingAddr")
			servingProto := viper.GetString("servingProto")
			port := viper.GetInt("port")
			bridge := viper.GetString("bridge")
			intf := viper.GetString("interface")
			ovsCliDir := viper.GetString("ovsCliDir")
			bridgeType := viper.GetString("bridgeType")
			p4rtbin := viper.GetString("p4rtbin")
			portMuxVsi := viper.GetInt("portMuxVsi")
			mode := viper.GetString("mode")
			daemonHostIp := viper.GetString("daemonHostIp")
			daemonIpuIp := viper.GetString("daemonIpuIp")
			daemonPort := viper.GetInt("daemonPort")

			log.Info("Initializing IPU plugin")
			log.WithFields(log.Fields{
				"servingAddr":  servingAddr,
				"servingProto": servingProto,
				"port":         port,
				"bridge":       bridge,
				"interface":    intf,
				"ovsCliDir":    ovsCliDir,
				"bridgeType":   bridgeType,
				"p4rtbin":      p4rtbin,
				"portMuxVsi":   portMuxVsi,
				"mode":         mode,
				"daemonHostIp": daemonHostIp,
				"daemonIpuIp":  daemonIpuIp,
				"daemonHost":   daemonPort,
			}).Info("Configurations")

			brCtlr, brType := getBridgeController(bridge, bridgeType, ovsCliDir)
			p4Client := p4rtclient.NewP4RtClient(p4rtbin, portMuxVsi, defaultP4BridgeName, brType)

			mgr := ipuplugin.NewIpuPlugin(port, brCtlr, p4Client, servingAddr, servingProto, bridge, intf, ovsCliDir, mode, daemonHostIp, daemonIpuIp, daemonPort)
			if err := mgr.Run(); err != nil {
				exitWithError(err, 4)
			}
		},
	}
)

func exitWithError(err error, exitCode int) {
	fmt.Fprintf(os.Stderr, "There was an error while executing %s: %s\n", cliName, err.Error())
	os.Exit(exitCode)
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		exitWithError(err, 1)
	}
}
func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&config.cfgFile, "config", "", "config file (default is /etc/ipu/ipuplugin.yaml)")

	rootCmd.PersistentFlags().StringVar(&config.logDir, "logDir", defaultLogDir, "IPU plugin log directory")
	rootCmd.PersistentFlags().IntVar(&config.port, "port", defaultServingPort, "IPU plugin serving TCP port")
	rootCmd.PersistentFlags().StringVar(&config.servingAddr, "servingAddr", defaultServingAddr, "IPU plugin serving host")
	rootCmd.PersistentFlags().StringVar(&config.servingProto, "servingProto", defaultServingProto, "IPU plugin serving protocol: 'unix|tcp'")
	rootCmd.PersistentFlags().StringVar(&config.interfaceName, "interface", "", "The uplink network interface name")
	rootCmd.PersistentFlags().StringVar(&config.bridge, "bridge", tenantBridgeName, "The bridge name that IPU plugin will manage")
	rootCmd.PersistentFlags().StringVar(&config.ovsCliDir, "ovsCliDir", defaultOvsCliDir, "The directory where the ovs-vsctl is located")
	rootCmd.PersistentFlags().StringVar(&config.bridgeType, "bridgeType", defaultBridge, "The bridge type that IPU plugin will manage")
	rootCmd.PersistentFlags().StringVar(&config.p4rtbin, "p4rtbin", defaultP4rtBin, "The directory where the p4rt-ctl binary is located")
	rootCmd.PersistentFlags().IntVar(&config.portMuxVsi, "portMuxVsi", defaultPortMuxVsi,
		"The port mux VSI number. This must be for the same interface from --interface flags")
	//Default Log level value is the warn level
	rootCmd.PersistentFlags().StringVarP(&config.verbosity, "verbosity", "v", log.InfoLevel.String(), "Log level (debug, info, warn, error, fatal, panic")
	rootCmd.PersistentFlags().StringVar(&config.mode, "mode", "", "IPU plugin daemon mode: host|ipu (required)")
	rootCmd.PersistentFlags().StringVar(&config.daemonHostIp, "daemonHostIp", defaultDaemonHostIp, "Daemon address on host")
	rootCmd.PersistentFlags().StringVar(&config.daemonIpuIp, "daemonIpuIp", defaultDaemonIpuIp, "Daemon address on ipu")
	rootCmd.PersistentFlags().IntVar(&config.daemonPort, "daemonPort", defaultDaemonPort, "Daemon port port")

	if err := rootCmd.MarkPersistentFlagRequired("mode"); err != nil {
		exitWithError(err, 1)
	}

	// Update below list of flags for any new flags added/updated above to bind them via Viper
	flagList := []string{
		"logDir",
		"port",
		"servingAddr",
		"servingProto",
		"interface",
		"bridge",
		"ovsCliDir",
		"bridgeType",
		"p4rtbin",
		"portMuxVsi",
		"verbosity",
		"mode",
		"daemonHostIp",
		"daemonIpuIp",
		"daemonPort",
	}

	for _, f := range flagList {
		if err := viper.BindPFlag(f, rootCmd.PersistentFlags().Lookup(f)); err != nil {
			fmt.Fprintf(os.Stderr, "There was an error while binding flags '%s'", err)
			os.Exit(1)
		}
	}
}

func initConfig() {
	if config.cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(config.cfgFile)
	} else {
		// Search config in default location
		viper.AddConfigPath(configFilePath)
		viper.SetConfigType("yaml")
		viper.SetConfigName("ipuplugin.yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func validateConfigs() error {
	if config.mode != "host" && config.mode != "ipu" {
		return fmt.Errorf("invalid mode specified: %s", config.mode)
	}
	return nil
}

func logInit(logDir string, logLevel string) error {
	if err := os.MkdirAll(logDir, 0644); err != nil {
		return err
	}

	logFilename := path.Join(logDir, cliName+".log")
	verifiedFileName, err := utils.VerifiedFilePath(logFilename, logDir)
	if err != nil {
		return err
	}

	logFile, err := os.OpenFile(verifiedFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	lgLvl, err := log.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	log.SetLevel(lgLvl)
	log.SetFormatter(&log.TextFormatter{
		PadLevelText:     true,
		QuoteEmptyFields: true,
	})
	if lgLvl == log.DebugLevel {
		log.SetReportCaller(true)
	}
	return nil
}

func getBridgeController(bridge, bridgeType, ovsCliDir string) (types.BridgeController, types.BridgeType) {
	switch bridgeType {
	case "ovs":
		return ipuplugin.NewOvsBridgeController(bridge, ovsCliDir), types.OvsBridge
	case "linux":
		return ipuplugin.NewLinuxBridgeController(bridge), types.LinuxBridge
	default:
		return ipuplugin.NewLinuxBridgeController(bridge), types.LinuxBridge
	}
}
