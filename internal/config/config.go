package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

const (
	reportSecDefault = 10
	pollSecDefault   = 2
)

// ServerConfig holds configuration for the HTTP server.
type ServerConfig struct {
	// Address is the listen address, e.g. "localhost:8080".
	Address string
}

// AgentConfig holds configuration for the metrics agent.
type AgentConfig struct {
	// Address is the HTTP server endpoint, host:port (no scheme).
	Address        string
	ReportInterval time.Duration
	PollInterval   time.Duration
}

// LoadServerConfigFromFlags parses CLI flags for the server binary.
// -a=<value> — listen address (default: localhost:8080).
func LoadServerConfigFromFlags() (*ServerConfig, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	cfg := &ServerConfig{}

	fs.StringVar(&cfg.Address, "a", "localhost:8080", "HTTP server listen address")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, err
	}
	// Reject leftover args (unknown positional args)
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}

	return cfg, nil
}

// LoadAgentConfigFromFlags parses CLI flags for the agent binary.
// -a=<value> — server endpoint address (default: localhost:8080).
// -r=<value> — report interval in seconds (default: 10).
// -p=<value> — poll interval in seconds (default: 2).
func LoadAgentConfigFromFlags() (*AgentConfig, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	cfg := &AgentConfig{}

	var reportSec int
	var pollSec int

	fs.StringVar(&cfg.Address, "a", "localhost:8080", "HTTP server endpoint address (host:port)")
	fs.IntVar(&reportSec, "r", reportSecDefault, "report interval in seconds")
	fs.IntVar(&pollSec, "p", pollSecDefault, "poll interval in seconds")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, err
	}
	// Reject leftover args (unknown positional args)
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}

	if reportSec <= 0 {
		return nil, fmt.Errorf("-r argument value must be greater then 0, provided: %v", reportSec)
	}
	if pollSec <= 0 {
		return nil, fmt.Errorf("-p argument value must be greater then 0, provided: %v", pollSec)
	}
	cfg.ReportInterval = time.Duration(reportSec) * time.Second
	cfg.PollInterval = time.Duration(pollSec) * time.Second

	return cfg, nil
}
