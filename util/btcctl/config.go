package main

import (
	"fmt"
	"github.com/conformal/btcutil"
	"github.com/conformal/go-flags"
	"os"
	"path/filepath"
	"strings"
)

var (
	btcdHomeDir        = btcutil.AppDataDir("btcd", false)
	btcctlHomeDir      = btcutil.AppDataDir("btcctl", false)
	defaultConfigFile  = filepath.Join(btcctlHomeDir, "btcctl.conf")
	defaultRPCCertFile = filepath.Join(btcdHomeDir, "rpc.cert")
)

// config defines the configuration options for btcctl.
//
// See loadConfig for details on the configuration load process.
type config struct {
	ShowVersion   bool   `short:"V" long:"version" description:"Display version information and exit"`
	ConfigFile    string `short:"C" long:"configfile" description:"Path to configuration file"`
	RPCUser       string `short:"u" long:"rpcuser" description:"RPC username"`
	RPCPassword   string `short:"P" long:"rpcpass" default-mask:"-" description:"RPC password"`
	RPCServer     string `short:"s" long:"rpcserver" description:"RPC server to connect to"`
	RPCCert       string `short:"c" long:"rpccert" description:"RPC server certificate chain for validation"`
	NoTls         bool   `long:"notls" description:"Disable TLS"`
	TlsSkipVerify bool   `long:"skipverify" description:"Do not verify tls certificates (not recommended!)"`
}

// cleanAndExpandPath expands environement variables and leading ~ in the
// passed path, cleans the result, and returns it.
func cleanAndExpandPath(path string) string {
	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		homeDir := filepath.Dir(btcctlHomeDir)
		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but they variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}

// loadConfig initializes and parses the config using a config file and command
// line options.
//
// The configuration proceeds as follows:
// 	1) Start with a default config with sane settings
// 	2) Pre-parse the command line to check for an alternative config file
// 	3) Load configuration file overwriting defaults with any specified options
// 	4) Parse CLI options and overwrite/add any specified options
//
// The above results in functioning properly without any config settings
// while still allowing the user to override settings with config files and
// command line options.  Command line options always take precedence.
func loadConfig() (*flags.Parser, *config, []string, error) {
	// Default config.
	cfg := config{
		ConfigFile: defaultConfigFile,
		RPCServer:  "localhost:8334",
		RPCCert:    defaultRPCCertFile,
	}

	// Create the home directory if it doesn't already exist.
	err := os.MkdirAll(btcdHomeDir, 0700)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(-1)
	}

	// Pre-parse the command line options to see if an alternative config
	// file or the version flag was specified.  Any errors can be ignored
	// here since they will be caught be the final parse below.
	preCfg := cfg
	preParser := flags.NewParser(&preCfg, flags.None)
	preParser.Parse()

	// Show the version and exit if the version flag was specified.
	if preCfg.ShowVersion {
		appName := filepath.Base(os.Args[0])
		appName = strings.TrimSuffix(appName, filepath.Ext(appName))
		fmt.Println(appName, "version", version())
		os.Exit(0)
	}

	// Load additional config from file.
	parser := flags.NewParser(&cfg, flags.PassDoubleDash|flags.HelpFlag)
	err = flags.NewIniParser(parser).ParseFile(preCfg.ConfigFile)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			fmt.Fprintln(os.Stderr, err)
			return parser, nil, nil, err
		}
	}

	// Parse command line options again to ensure they take precedence.
	remainingArgs, err := parser.Parse()
	if err != nil {
		return parser, nil, nil, err
	}

	// Handle environment variable expansion in the RPC certificate path.
	cfg.RPCCert = cleanAndExpandPath(cfg.RPCCert)

	return parser, &cfg, remainingArgs, nil
}
