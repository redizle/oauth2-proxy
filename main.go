package main

import (
	"fmt"
	"os"

	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/apis/options"
	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/validation"
	"github.com/spf13/pflag"
)

func main() {
	log := logger.NewLogEntry()

	flagSet := pflag.NewFlagSet("oauth2-proxy", pflag.ContinueOnError)

	// Define configuration flags
	config := flagSet.String("config", "", "path to config file")
	showVersion := flagSet.Bool("version", false, "print version string")
	alphaConfig := flagSet.String("alpha-config", "", "path to alpha config file (experimental)")

	// Add legacy flags for backwards compatibility
	options.RegisterLegacyFlagSetFlags(flagSet)

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *showVersion {
		fmt.Printf("oauth2-proxy %s (built with %s)\n", VERSION, runtime.Version())
		return
	}

	// Load configuration
	opts, err := loadConfiguration(*config, *alphaConfig, flagSet, os.Args[1:])
	if err != nil {
		log.Fatalf("ERROR: failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := validation.Validate(opts); err != nil {
		log.Fatalf("ERROR: invalid configuration: %v", err)
	}

	// Create and run the proxy
	oauthProxy, err := proxy.NewOAuthProxy(opts, func(email string) bool {
		return opts.IsValidatedEmail(email)
	})
	if err != nil {
		log.Fatalf("ERROR: failed to create oauth proxy: %v", err)
	}

	server := server.NewServer(opts, oauthProxy)
	if err := server.Start(); err != nil {
		log.Fatalf("ERROR: server failed: %v", err)
	}
}

// loadConfiguration reads and merges configuration from file and flags.
// Priority order (highest to lowest): CLI flags > alpha config > legacy config file
func loadConfiguration(configFile, alphaConfigFile string, flagSet *pflag.FlagSet, args []string) (*options.Options, error) {
	opts := options.NewOptions()

	// Load from config file if provided
	if configFile != "" {
		if err := options.LoadConfig(configFile, opts); err != nil {
			return nil, fmt.Errorf("loading config file %q: %w", configFile, err)
		}
	}

	// Load alpha config if provided (takes precedence over legacy config)
	if alphaConfigFile != "" {
		alphaCfg, err := options.LoadAlphaOptions(alphaConfigFile)
		if err != nil {
			return nil, fmt.Errorf("loading alpha config file %q: %w", alphaConfigFile, err)
		}
		options.ApplyAlphaOptions(opts, alphaCfg)
	}

	// Override with any flags explicitly set on command line
	if err := options.ApplyFlagSetToOptions(flagSet, opts); err != nil {
		return nil, fmt.Errorf("applying flags: %w", err)
	}

	return opts, nil
}
