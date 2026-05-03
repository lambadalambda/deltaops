package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	pathruntime "runtime"
	"strings"

	"deltaops/internal/collector"
	"deltaops/internal/config"
	appruntime "deltaops/internal/runtime"
)

const (
	exitOK      = 0
	exitStartup = 2
)

type Options struct {
	Stdout io.Writer
	Stderr io.Writer
	Env    map[string]string
	GOOS   string

	Version string
	Commit  string
}

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
			NextAction: "run DeltaOps on linux for the MVP collector set",
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

	return printStartupError(options.Stderr, missingRPCHelperError(options.GOOS))
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

func missingRPCHelperError(goos string) *appruntime.OperatorError {
	return &appruntime.OperatorError{
		Message: "Delta Chat RPC helper is not packaged in this build",
		NextAction: fmt.Sprintf(
			"build a %s/%s release with an embedded deltachat-rpc-server asset before running the monitor",
			goos,
			pathruntime.GOARCH,
		),
	}
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
	if options.Version == "" {
		options.Version = "dev"
	}
	return options
}
