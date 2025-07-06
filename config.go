package main

import (
	"time"
)

type Config struct {
	FilePath  string
	ProblemID string
	Timeout   string
	Verbose   bool
	CacheDir  string
	Parallel  int
	ShowDiff  bool
	MaxOutput int
	Optimize  bool
	Race      bool
	ForceAuth bool
}

func (c *Config) GetTimeout() time.Duration {
	duration, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 1 * time.Second // default
	}
	return duration
}

func (c *Config) GetBuildFlags() []string {
	var flags []string

	if c.Optimize {
		flags = append(flags, "-ldflags", "-s -w")
	}

	if c.Race {
		flags = append(flags, "-race")
	}

	return flags
}

func (c *Config) GetAuthCacheDir() string {
	return c.CacheDir + "/.auth"
}

func (c *Config) GetSessionFile() string {
	return c.GetAuthCacheDir() + "/session.json"
}
