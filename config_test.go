package main

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalAutoScaleGroup(t *testing.T) {
	raw := `
{
  "App": "hello",
  "MemoryHigh": "100m",
  "MemoryLow": 102400,
  "Service": "web",
  "NotExists":true,
  "MaxContainers": 2,
  "MinContainers": 1
}`
	c := &AutoScaleGroup{}
	if err := json.Unmarshal([]byte(raw), c); err != nil {
		t.Fatalf("Failed to unmarshal Config: %v", err)
	}

	if c.MemoryLow != 102400 {
		t.Fatalf("unexpected c.MemoryLow. expected %d, got %d", 1024000, c.MemoryLow)
	}

	mh := float64(1000 * 1000 * 100)
	if c.MemoryHigh != mh {
		t.Fatalf("unexpected c.MemoryHigh. expected %d, got %d", mh, c.MemoryHigh)
	}

	if c.App != "hello" {
		t.Fatalf("expected c.App == hello, got %s", c.App)
	}
}

func TestParseConfig(t *testing.T) {
	cfg, err := ParseConfig("sample-config.json")
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if cfg.ApiKey == "" {
		t.Fatal("expected cfg.ApiKey to be not empty")
	}

	if len(cfg.Groups) < 0 {
		t.Fatal("expecte cfg.Groups to contain at least 1 items")
	}
}
