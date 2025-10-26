package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	reportSecDefault       = 10
	pollSecDefault         = 2
	storeIntervaleDefault  = 300
	FileStoragePathDefault = "/tmp/metrics-db.json"
)

// ServerConfig holds configuration for the HTTP server.
type ServerConfig struct {
	// Address is the listen address, e.g. "localhost:8080".
	Address         string
	StoreIntervale  time.Duration
	FileStoragePath string
	Restore         bool
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

	var storeSec int

	fs.StringVar(&cfg.Address, "a", "localhost:8080", "HTTP server listen address")
	fs.IntVar(&storeSec, "i", storeIntervaleDefault, "store interval in seconds")
	fs.StringVar(&cfg.FileStoragePath, "f", FileStoragePathDefault, "full filename for storage file")
	fs.BoolVar(&cfg.Restore, "r", true, "restore values on start")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, err
	}
	// Reject leftover args (unknown positional args)
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	// Env override: ADDRESS has highest priority
	if addr, ok := os.LookupEnv("ADDRESS"); ok && addr != "" {
		cfg.Address = addr
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("invalid STORE_INTERVAL, must be positive integer seconds or 0: %q", v)
		}
		storeSec = n
	}
	if filePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok && filePath != "" {
		cfg.FileStoragePath = filePath
	}
	if restoreVal, ok := os.LookupEnv("RESTORE"); ok && restoreVal != "" {
		switch restoreVal {
		case "true":
			cfg.Restore = true
		case "false":
			cfg.Restore = false
		default:
			return nil, fmt.Errorf("invalid RESTORE, must be true or false: %q", restoreVal)
		}
	}

	cfg.StoreIntervale = time.Duration(storeSec) * time.Second

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

	// Env overrides have highest priority
	if addr, ok := os.LookupEnv("ADDRESS"); ok && addr != "" {
		cfg.Address = addr
	}
	if v, ok := os.LookupEnv("REPORT_INTERVAL"); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid REPORT_INTERVAL, must be positive integer seconds: %q", v)
		}
		reportSec = n
	}
	if v, ok := os.LookupEnv("POLL_INTERVAL"); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid POLL_INTERVAL, must be positive integer seconds: %q", v)
		}
		pollSec = n
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
