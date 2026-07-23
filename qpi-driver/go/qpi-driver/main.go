// Command qpi-driver is the CLI that runs QPI's officially maintained Go
// built-in drivers, mirroring the Python `qpi-driver` CLI (RFC 0001 §4): the
// operation is the subcommand (process | monitor), the device selects the
// backend within it, universal flags are shared, and a device's own settings
// are passed as repeatable `-o key=value`.
//
// Install it with:
//
//	go install github.com/sopherapps/qpi/qpi-driver/go/qpi-driver@latest
//
// then, e.g.:
//
//	qpi-driver monitor --device bluefors_gen1 \
//	  --qpi-addr https://qpi.example.com --token … --ca-fingerprint … \
//	  -o base_url=http://localhost:49099 -o channels=mapper.bf.tmc:K
package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	qpidriver "github.com/sopherapps/qpi/qpi-driver/go"
	"github.com/sopherapps/qpi/qpi-driver/go/qpi-driver/bluefors"
)

// version is the CLI version; overridable at build time with
// -ldflags "-X main.version=…".
var version = "0.1.0"

// commonFlags are the universal options every operation subcommand shares,
// mirroring the Python CLI. A device's own settings go through -o instead.
type commonFlags struct {
	qpiAddr       string
	token         string
	name          string
	device        string
	caFile        string
	caFingerprint string
	options       []string
	recvTimeoutMs int
}

// deviceRunner runs one device of an operation from the shared flags and the
// parsed -o options; it blocks until the driver is stopped.
type deviceRunner func(cf *commonFlags, opts map[string]string) error

// processDrivers and monitorDrivers map a device to its runner. Go ships no
// process (QPU) built-in yet, so that registry is empty; both operations are
// dispatched the same way, so adding a device is one entry here.
var (
	processDrivers = map[string]deviceRunner{}
	monitorDrivers = map[string]deviceRunner{
		"bluefors_gen1": runBlueforsGen1,
	}
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "qpi-driver",
		Short:         "Quantum Processing Interface (QPI) Driver CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newOperationCmd("process", "mock", "qpu_sim_01",
			"Run a process driver — a QPU that executes jobs pushed to it (RFC 0001 §4).",
			processDrivers),
		newOperationCmd("monitor", "bluefors_gen1", "qpi-monitor",
			"Run a monitor driver — one that only reports upward on its own schedule (RFC 0001 §7).",
			monitorDrivers),
		newVersionCmd(),
	)
	return root
}

func newOperationCmd(operation, defaultDevice, defaultName, short string, registry map[string]deviceRunner) *cobra.Command {
	cf := &commonFlags{}
	cmd := &cobra.Command{
		Use:   operation,
		Short: short,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runOperation(operation, cf, registry)
		},
	}
	addCommonFlags(cmd, cf, defaultDevice, defaultName)
	return cmd
}

func addCommonFlags(cmd *cobra.Command, cf *commonFlags, defaultDevice, defaultName string) {
	f := cmd.Flags()
	f.StringVarP(&cf.qpiAddr, "qpi-addr", "a", envOr("QPI_ADDR", "http://127.0.0.1:8090"),
		"Full URL of the QPI server")
	f.StringVarP(&cf.token, "token", "t", os.Getenv("QPI_ACCESS_TOKEN"),
		"Access token identifying this driver to the QPI server")
	f.StringVarP(&cf.name, "name", "n", envOr("QPI_DRIVER_NAME", defaultName),
		"Human-readable name for this driver")
	f.StringVarP(&cf.device, "device", "d", envOr("QPI_DEVICE", defaultDevice),
		"Which backend to run within the operation")
	f.StringVar(&cf.caFile, "ca-file", envOr("QPI_CA_FILE", "./bin/qpi.ca.pem"),
		"Where the downloaded server root CA certificate is written")
	f.StringVar(&cf.caFingerprint, "ca-fingerprint", os.Getenv("QPI_CA_FINGERPRINT"),
		"SHA-256 fingerprint pinning the downloaded root CA of the QPI server")
	f.StringArrayVarP(&cf.options, "option", "o", nil,
		"Operation-specific config as key=value, repeatable (e.g. -o channels=mapper.bf.tmc:K)")
	f.IntVar(&cf.recvTimeoutMs, "recv-timeout-ms",
		envIntOr("QPI_RECV_TIMEOUT_MS", int(qpidriver.DefaultRecvTimeout/time.Millisecond)),
		"How long the receive loop blocks per attempt before checking for shutdown, in ms")
}

// runOperation looks up the device's runner in the operation's registry and
// runs it, mirroring the Python CLI's shared operation handler.
func runOperation(operation string, cf *commonFlags, registry map[string]deviceRunner) error {
	if cf.token == "" {
		return fmt.Errorf("access token is required; set --token/-t or the QPI_ACCESS_TOKEN environment variable")
	}
	runner, ok := registry[cf.device]
	if !ok {
		return fmt.Errorf("unknown %s device %q; known devices: %s",
			operation, cf.device, strings.Join(knownDevices(registry), ", "))
	}
	opts, err := parseOptions(cf.options)
	if err != nil {
		return err
	}
	return runner(cf, opts)
}

// runBlueforsGen1 builds the Bluefors Gen. 1 monitor from the -o options and
// runs it. Recognised keys mirror the Python driver: channels (required),
// base_url, api_key, poll_interval (seconds), timeout (seconds).
func runBlueforsGen1(cf *commonFlags, opts map[string]string) error {
	channels := opts["channels"]
	if channels == "" {
		return fmt.Errorf("bluefors_gen1 needs a 'channels' option, e.g. " +
			"-o channels=mapper.bf.tmc:K,mapper.bf.pmc:mbar")
	}
	monitor := bluefors.New(bluefors.Options{
		BaseURL:      optOr(opts, "base_url", bluefors.DefaultBaseURL),
		Channels:     bluefors.ParseChannels(channels),
		APIKey:       opts["api_key"],
		PollInterval: secondsOr(opts, "poll_interval", bluefors.DefaultPollInterval),
		Timeout:      secondsOr(opts, "timeout", bluefors.DefaultTimeout),
	})
	return qpidriver.Run(monitor, qpidriver.Config{
		QpiAddr:       cf.qpiAddr,
		Token:         cf.token,
		Name:          cf.name,
		CaFingerprint: cf.caFingerprint,
		CaFilePath:    cf.caFile,
		RecvTimeout:   time.Duration(cf.recvTimeoutMs) * time.Millisecond,
	})
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the version of the QPI driver CLI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), version)
			return nil
		},
	}
}

// parseOptions turns repeatable `-o key=value` flags into a dict; each device
// reads the keys it cares about, keeping the CLI generic.
func parseOptions(pairs []string) (map[string]string, error) {
	opts := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid option %q; expected key=value", pair)
		}
		opts[key] = strings.TrimSpace(value)
	}
	return opts, nil
}

func knownDevices(registry map[string]deviceRunner) []string {
	devices := make([]string, 0, len(registry))
	for device := range registry {
		devices = append(devices, device)
	}
	sort.Strings(devices)
	return devices
}

func optOr(opts map[string]string, key, fallback string) string {
	if v, ok := opts[key]; ok && v != "" {
		return v
	}
	return fallback
}

func secondsOr(opts map[string]string, key string, fallback time.Duration) time.Duration {
	if v, ok := opts[key]; ok && v != "" {
		if secs, err := strconv.ParseFloat(v, 64); err == nil {
			return time.Duration(secs * float64(time.Second))
		}
	}
	return fallback
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
