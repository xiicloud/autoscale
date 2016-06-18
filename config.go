package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"reflect"
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
	MaxContainers float64
	MinContainers float64
}

type Config struct {
	Groups         []*AutoScaleGroup
	ControllerAddr string
	ApiKey         string
}

func (c *AutoScaleGroup) UnmarshalJSON(b []byte) error {
	m := make(map[string]interface{})
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	if v, ok := m["App"].(string); !ok {
		return fmt.Errorf("App must be a string")

	} else {
		c.App = v
	}

	if v, ok := m["Service"].(string); !ok {
		return fmt.Errorf("Service must be a string")
	} else {
		c.Service = v
	}

	fields := map[string]bool{
		"CpuHigh":       true,
		"CpuLow":        true,
		"MemoryHigh":    true,
		"MemoryLow":     true,
		"MaxContainers": true,
		"MinContainers": true,
	}

	vc := reflect.Indirect(reflect.ValueOf(c))
	for k, v := range m {
		if !fields[k] {
			continue
		}

		f := vc.FieldByName(k)
		switch t := v.(type) {
		case string:
			vv, err := units.FromHumanSize(t)
			if err != nil {
				return fmt.Errorf("failed to parse %s to int: %v", k, err)
			}

			f.SetFloat(float64(vv))
		case float64:
			f.SetFloat(t)

		default:
			xx := reflect.TypeOf(t)
			return fmt.Errorf("%s must be an integer, got %s", k, xx.Name())
		}
	}

	return nil
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
