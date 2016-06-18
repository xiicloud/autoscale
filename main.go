package main

import (
	"os"
	"sync"

	"github.com/Sirupsen/logrus"
)

func main() {
	path := configPath
	if os.Getenv("DEBUG") == "true" {
		logrus.SetLevel(logrus.DebugLevel)
		if _, err := os.Stat("./sample-config.json"); err == nil {
			path = "./sample-config.json"
		}
	}

	cfg, err := ParseConfig(path)
	if err != nil {
		logrus.Fatalf("Invalid config: %v", err)
	}
	checkConfig(cfg)

	run(cfg)
}

func run(cfg *Config) {
	var wg sync.WaitGroup
	wg.Add(len(cfg.Groups))
	for _, g := range cfg.Groups {
		m := newMonitor(g)
		go func() {
			m.start()
			wg.Done()
		}()
	}

	wg.Wait()
}
