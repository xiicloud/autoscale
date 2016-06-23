package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/go-units"
)

const (
	configPath = "/etc/autoscale.json"
)

var (
	controllerAddr = os.Getenv("CONTROLLER_ADDR")
	apiKey         = os.Getenv("API_KEY")
)

func checkConfig(cfg *Config) {
	// TODO: remove these global vars
	if cfg.ControllerAddr != "" {
		controllerAddr = cfg.ControllerAddr
	}
	if cfg.ApiKey != "" {
		apiKey = cfg.ApiKey
	}

	if controllerAddr == "" {
		logrus.Fatal("ControllerAddr must be provided")
	}
	if _, err := url.Parse(controllerAddr); err != nil {
		logrus.Fatal("ControllerAddr is not a valid URL")
	}
	controllerAddr = strings.TrimRight(controllerAddr, "/")

	if apiKey == "" {
		logrus.Fatal("ApiKey must be provided")
	}
}

type AutoScaleGroup struct {
	App           string
	Service       string
	CpuHigh       float64
	CpuLow        float64
	MemoryHigh    float64
	MemoryLow     float64
	MaxContainers int
	MinContainers int
}

type Config struct {
	Groups         []*AutoScaleGroup
	ControllerAddr string
	ApiKey         string
}

func (c *AutoScaleGroup) UnmarshalJSON(b []byte) error {
	type tmp AutoScaleGroup // alias to avoid endless recursion.
	m := struct {
		tmp
		MemoryHigh interface{}
		MemoryLow  interface{}
	}{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	if m.MaxContainers < 1 {
		return fmt.Errorf("MaxContainers must be a positive integer")
	}

	if m.MinContainers < 1 {
		return fmt.Errorf("MinContainers must be a positive integer")
	} else if m.MinContainers > m.MaxContainers {
		return fmt.Errorf("MinContainers must be less than MaxContainers")
	}

	var err error
	if m.tmp.MemoryHigh, err = parseFloat64(m.MemoryHigh); err != nil {
		return err
	}

	if m.tmp.MemoryLow, err = parseFloat64(m.MemoryLow); err != nil {
		return err
	}

	*c = AutoScaleGroup(m.tmp)
	return nil
}

func parseFloat64(v interface{}) (float64, error) {
	switch t := v.(type) {
	case string:
		vv, err := units.FromHumanSize(t)
		if err != nil {
			return 0, fmt.Errorf("failed to parse %v to int: %v", v, err)
		}
		return float64(vv), nil
	case float64:
		return t, nil
	default:
		return 0, fmt.Errorf("%v must be an integer", v)
	}
}

func ParseConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()
	cfg := &Config{}
	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
