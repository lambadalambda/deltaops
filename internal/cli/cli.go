package cli

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	pathruntime "runtime"
	"strings"

	"deltaops/internal/alert"
	"deltaops/internal/binding"
	"deltaops/internal/collector"
	"deltaops/internal/config"
	"deltaops/internal/notify"
	"deltaops/internal/notify/dcrpc"
	appruntime "deltaops/internal/runtime"
)

const (
	exitOK      = 0
	exitStartup = 2
)

type Options struct {
	Stdout         io.Writer
	Stderr         io.Writer
	Env            map[string]string
	GOOS           string
	GOARCH         string
	RuntimeFactory RuntimeFactory

	Version string
	Commit  string
}

type RuntimeConfig struct {
	Paths        config.Paths
	Provisioning config.DeltaChatProvisioning
	GOOS         string
	GOARCH       string
	Stdout       io.Writer
	Stderr       io.Writer
}

type RuntimeProcess interface {
	Run(context.Context) error
	Close() error
}

type RuntimeFactory func(context.Context, RuntimeConfig) (RuntimeProcess, error)

func Run(args []string, options Options) int {
	options = options.withDefaults()
	if len(args) == 0 {
		return runCommand(nil, options)
	}

	command := args[0]
	switch command {
	case "run":
		return runCommand(args[1:], options)
	case "version":
		return versionCommand(args[1:], options)
	case "help", "--help", "-h":
		printUsage(options.Stdout)
		return exitOK
	default:
		return printStartupError(options.Stderr, &appruntime.OperatorError{
			Message:    "unknown command",
			NextAction: "run deltaops --help to list commands",
		})
	}
}

func runCommand(args []string, options Options) int {
	flags := flag.NewFlagSet("deltaops run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var dcAccountURL string
	var configPath string
	var stateDir string
	flags.StringVar(&dcAccountURL, "dcaccount-url", "", "chatmail dcaccount URL used to provision the Delta Chat account")
	flags.StringVar(&configPath, "config", "", "path to config file")
	flags.StringVar(&stateDir, "state-dir", "", "path to DeltaOps state directory")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printRunUsage(options.Stdout)
			return exitOK
		}
		return printStartupError(options.Stderr, &appruntime.OperatorError{
			Message:    "invalid run arguments",
			NextAction: "run deltaops run --help for supported flags",
			Cause:      err,
		})
	}
	if flags.NArg() != 0 {
		return printStartupError(options.Stderr, &appruntime.OperatorError{
			Message:    "unexpected run argument",
			NextAction: "run deltaops run --help for supported flags",
		})
	}

	if err := collector.ValidatePlatform(options.GOOS); err != nil {
		return printStartupError(options.Stderr, &appruntime.OperatorError{
			Message:    "unsupported operating system",
			NextAction: "run DeltaOps on linux or macOS development mode",
			Cause:      err,
		})
	}

	flagDCAccountURL := strings.TrimSpace(dcAccountURL)
	envDCAccountURL := envValue(options, config.DCAccountURLEnv)
	needConfig := flagDCAccountURL == "" && strings.TrimSpace(envDCAccountURL) == ""
	paths, err := resolveRunPaths(options, configPath, stateDir, needConfig)
	if err != nil {
		return printStartupError(options.Stderr, &appruntime.OperatorError{
			Message:    "cannot resolve config and state paths",
			NextAction: "set HOME, XDG_CONFIG_HOME, XDG_STATE_HOME, --config, or --state-dir to absolute paths",
			Cause:      err,
		})
	}

	var configDCAccountURL string
	if needConfig {
		configDCAccountURL, err = readConfigDCAccountURL(paths.ConfigPath, strings.TrimSpace(configPath) != "")
		if err != nil {
			return printStartupError(options.Stderr, &appruntime.OperatorError{
				Message:    "cannot read config file",
				NextAction: "fix the config path or provide --dcaccount-url or DELTAOPS_DCACCOUNT_URL",
				Cause:      err,
			})
		}
	}

	startup := config.StartupConfig{Provisioning: config.ResolveProvisioning(config.ProvisioningSources{
		FlagDCAccountURL:   flagDCAccountURL,
		EnvDCAccountURL:    envDCAccountURL,
		ConfigDCAccountURL: configDCAccountURL,
	})}
	if err := startup.Validate(); err != nil {
		return printStartupError(options.Stderr, &appruntime.OperatorError{
			Message:    "startup configuration is incomplete",
			NextAction: "provide --dcaccount-url, DELTAOPS_DCACCOUNT_URL, or delta_chat.dcaccount_url",
			Cause:      err,
		})
	}

	if err := config.EnsureStateLayout(paths); err != nil {
		return printStartupError(options.Stderr, &appruntime.OperatorError{
			Message:    "cannot prepare state directory",
			NextAction: "choose a writable private state directory with --state-dir or XDG_STATE_HOME",
			Cause:      err,
		})
	}

	runtimeConfig := RuntimeConfig{
		Paths:        paths,
		Provisioning: startup.Provisioning,
		GOOS:         options.GOOS,
		GOARCH:       options.GOARCH,
		Stdout:       options.Stdout,
		Stderr:       options.Stderr,
	}
	process, err := options.RuntimeFactory(context.Background(), runtimeConfig)
	if err != nil {
		return printStartupError(options.Stderr, operatorErrorOrDefault(err, "cannot start DeltaOps runtime", "check Delta Chat helper packaging, account state, and local logs"))
	}
	defer process.Close() //nolint:errcheck
	if err := process.Run(context.Background()); err != nil {
		return printStartupError(options.Stderr, operatorErrorOrDefault(err, "DeltaOps runtime stopped with an error", "check Delta Chat account state, host metrics, and local logs"))
	}
	return exitOK
}

func versionCommand(args []string, options Options) int {
	if len(args) != 0 {
		return printStartupError(options.Stderr, &appruntime.OperatorError{
			Message:    "unexpected version argument",
			NextAction: "run deltaops version without arguments",
		})
	}
	if options.Commit == "" {
		fmt.Fprintf(options.Stdout, "deltaops %s\n", options.Version)
		return exitOK
	}
	fmt.Fprintf(options.Stdout, "deltaops %s (%s)\n", options.Version, options.Commit)
	return exitOK
}

func missingRPCHelperError(goos, goarch string) *appruntime.OperatorError {
	return &appruntime.OperatorError{
		Message: "Delta Chat RPC helper is not packaged in this build",
		NextAction: fmt.Sprintf(
			"build a %s/%s release with an embedded deltachat-rpc-server asset before running the monitor",
			goos,
			goarch,
		),
	}
}

func defaultRuntimeFactory(ctx context.Context, runtimeConfig RuntimeConfig) (RuntimeProcess, error) {
	transport, err := dcrpc.Open(ctx, dcrpc.Options{
		GOOS:         runtimeConfig.GOOS,
		GOARCH:       runtimeConfig.GOARCH,
		AccountsDir:  runtimeConfig.Paths.DeltaChatAccountsDir,
		HelperDir:    filepath.Join(runtimeConfig.Paths.StateDir, "deltachat-rpc-helper"),
		DCAccountURL: runtimeConfig.Provisioning.DCAccountURL,
		Stderr:       io.Discard,
		Assets:       dcrpc.EmbeddedHelpers(),
	})
	if err != nil {
		if errors.Is(err, dcrpc.ErrUnsupportedHelperTarget) {
			return nil, &appruntime.OperatorError{
				Message:    "Delta Chat RPC helper target is unsupported",
				NextAction: "run a build with one of these embedded helper targets: " + strings.Join(dcrpc.SupportedHelperTargets(), ", "),
				Cause:      err,
			}
		}
		if errors.Is(err, dcrpc.ErrHelperUnavailable) {
			return nil, missingRPCHelperError(runtimeConfig.GOOS, runtimeConfig.GOARCH)
		}
		return nil, &appruntime.OperatorError{Message: "cannot start Delta Chat transport", NextAction: "check Delta Chat helper packaging and local account state", Cause: err}
	}
	process, err := newRuntimeProcess(ctx, runtimeConfig, transport)
	if err != nil {
		_ = transport.Close()
		return nil, err
	}
	return process, nil
}

func newRuntimeProcess(ctx context.Context, runtimeConfig RuntimeConfig, transport *dcrpc.Transport) (RuntimeProcess, error) {
	setupCode, err := newSetupCode()
	if err != nil {
		return nil, &appruntime.OperatorError{Message: "cannot create pairing setup code", NextAction: "retry startup and check local randomness sources", Cause: err}
	}
	manager, err := binding.NewManager(setupCode, binding.NewFileStore(runtimeConfig.Paths.BindingPath))
	if err != nil {
		return nil, &appruntime.OperatorError{Message: "cannot load contact binding", NextAction: "inspect or reset the binding file in the DeltaOps state directory", Cause: err}
	}
	if _, ok := manager.BoundContact(); !ok {
		account, err := transport.Account(ctx)
		if err != nil {
			return nil, &appruntime.OperatorError{Message: "cannot read Delta Chat account contact data", NextAction: "check Delta Chat account setup and local state", Cause: err}
		}
		printPairingSetup(runtimeConfig.Stdout, account, setupCode)
	}

	collector, err := collector.NewCollector(runtimeConfig.GOOS, collector.Dependencies{}, nil)
	if err != nil {
		return nil, &appruntime.OperatorError{Message: "cannot create host metric collector", NextAction: "run DeltaOps on linux or macOS development mode", Cause: err}
	}
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		host = "unknown"
	}
	signals := appruntime.NewOSSignalSource()
	runner, err := appruntime.NewRunner(appruntime.Config{}, appruntime.Dependencies{
		Account:   notify.RuntimeAccount{Transport: transport},
		Pairer:    notify.RuntimePairer{Manager: manager, Transport: transport},
		Collector: collector,
		Evaluator: alert.NewEvaluator(alert.DefaultConfig(host), nil),
		Notifier:  notify.RuntimeNotifier{Transport: transport},
		Signals:   signals,
		Logger:    appruntime.NewJSONLogger(runtimeConfig.Stderr),
	})
	if err != nil {
		signals.Stop()
		return nil, &appruntime.OperatorError{Message: "cannot create DeltaOps runtime", NextAction: "check runtime configuration and local logs", Cause: err}
	}
	return &runtimeProcess{runner: runner, transport: transport, signals: signals}, nil
}

type runtimeProcess struct {
	runner    *appruntime.Runner
	transport *dcrpc.Transport
	signals   *appruntime.OSSignalSource
}

func (p *runtimeProcess) Run(ctx context.Context) error {
	return p.runner.Run(ctx)
}

func (p *runtimeProcess) Close() error {
	if p.signals != nil {
		p.signals.Stop()
	}
	if p.transport != nil {
		return p.transport.Close()
	}
	return nil
}

func newSetupCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func printPairingSetup(w io.Writer, account notify.Account, setupCode string) {
	fmt.Fprintln(w, "DeltaOps pairing setup")
	if strings.TrimSpace(account.ContactURI) != "" {
		fmt.Fprintf(w, "bot contact: %s\n", account.ContactURI)
	}
	if strings.TrimSpace(account.Address) != "" {
		fmt.Fprintf(w, "bot address: %s\n", account.Address)
	}
	fmt.Fprintf(w, "setup code: %s\n", setupCode)
}

func readConfigDCAccountURL(path string, explicit bool) (string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && !explicit {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return parseConfigDCAccountURL(string(data)), nil
}

func resolveRunPaths(options Options, configPathOverride, stateDirOverride string, needConfig bool) (config.Paths, error) {
	env := pathEnv(options)
	stateDir, err := config.ResolveStateDir(env, stateDirOverride)
	if err != nil {
		return config.Paths{}, err
	}
	configPath := strings.TrimSpace(configPathOverride)
	if needConfig {
		configPath, err = config.ResolveConfigPath(env, configPathOverride)
		if err != nil {
			return config.Paths{}, err
		}
	}
	return config.NewPaths(configPath, stateDir), nil
}

func parseConfigDCAccountURL(data string) string {
	inDeltaChat := false
	deltaChatIndent := 0
	for _, line := range strings.Split(data, "\n") {
		line = stripInlineComment(line)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if value, ok := parseKeyValue(trimmed, config.DCAccountURLConfigKey); ok {
			return value
		}
		if value, ok := parseKeyValue(trimmed, "delta_chat"); ok && value == "" {
			inDeltaChat = true
			deltaChatIndent = indent
			continue
		}
		if inDeltaChat && indent > deltaChatIndent {
			if value, ok := parseKeyValue(trimmed, "dcaccount_url"); ok {
				return value
			}
			continue
		}
		if inDeltaChat && indent <= deltaChatIndent {
			inDeltaChat = false
		}
	}
	return ""
}

func stripInlineComment(line string) string {
	inSingleQuote := false
	inDoubleQuote := false
	for i, r := range line {
		switch r {
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		case '#':
			if !inSingleQuote && !inDoubleQuote && (i == 0 || line[i-1] == ' ' || line[i-1] == '\t') {
				return line[:i]
			}
		}
	}
	return line
}

func parseKeyValue(line, key string) (string, bool) {
	prefix := key + ":"
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	return unquoteConfigValue(strings.TrimSpace(strings.TrimPrefix(line, prefix))), true
}

func unquoteConfigValue(value string) string {
	if len(value) < 2 {
		return strings.TrimSpace(value)
	}
	first := value[0]
	last := value[len(value)-1]
	if (first == '\'' && last == '\'') || (first == '"' && last == '"') {
		return strings.TrimSpace(value[1 : len(value)-1])
	}
	return strings.TrimSpace(value)
}

func pathEnv(options Options) config.PathEnv {
	return config.PathEnv{
		HomeDir:       envValue(options, "HOME"),
		XDGConfigHome: envValue(options, "XDG_CONFIG_HOME"),
		XDGStateHome:  envValue(options, "XDG_STATE_HOME"),
	}
}

func envValue(options Options, key string) string {
	if options.Env != nil {
		return options.Env[key]
	}
	return os.Getenv(key)
}

func printStartupError(w io.Writer, err error) int {
	fmt.Fprintf(w, "startup failed: %v\n", err)
	return exitStartup
}

func operatorErrorOrDefault(err error, message, nextAction string) error {
	var operatorErr *appruntime.OperatorError
	if errors.As(err, &operatorErr) {
		return operatorErr
	}
	return &appruntime.OperatorError{Message: message, NextAction: nextAction, Cause: err}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: deltaops [run|version]")
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  run      start the monitor with safe defaults")
	fmt.Fprintln(w, "  version  print version metadata")
}

func printRunUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: deltaops run [--dcaccount-url dcaccount:...] [--config path] [--state-dir path]")
}

func (options Options) withDefaults() Options {
	if options.Stdout == nil {
		options.Stdout = os.Stdout
	}
	if options.Stderr == nil {
		options.Stderr = os.Stderr
	}
	if options.GOOS == "" {
		options.GOOS = pathruntime.GOOS
	}
	if options.GOARCH == "" {
		options.GOARCH = pathruntime.GOARCH
	}
	if options.RuntimeFactory == nil {
		options.RuntimeFactory = defaultRuntimeFactory
	}
	if options.Version == "" {
		options.Version = "dev"
	}
	return options
}
