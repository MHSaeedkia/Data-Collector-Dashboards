package router

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// ChannelConfig holds the settings of one delivery channel (Telegram, Bale, ...).
// Extra fields (such as proxy_url) are stored in Extra so each sender reads the
// fields it needs itself, without polluting the shared struct.
type ChannelConfig struct {
	Type  string            `yaml:"type"`
	Extra map[string]string `yaml:",inline"`
}

// UnmarshalYAML is custom: it pulls out type and dumps the remaining keys into Extra.
func (c *ChannelConfig) UnmarshalYAML(value *yaml.Node) error {
	raw := map[string]string{}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	c.Extra = map[string]string{}
	for k, v := range raw {
		if k == "type" {
			c.Type = v
			continue
		}
		c.Extra[k] = v
	}
	return nil
}

// Rule is one routing rule: if all of Match's labels match the alert,
// the alert is sent to Channels.
type Rule struct {
	Name     string            `yaml:"name"`
	Match    map[string]string `yaml:"match"`
	Channels []string          `yaml:"channels"`
}

// Config is the whole structure of config.yaml.
type Config struct {
	Routes          []Rule                   `yaml:"routes"`
	DefaultChannels []string                 `yaml:"default_channels"`
	Channels        map[string]ChannelConfig `yaml:"channels"`
}

// envVarPattern finds variables of the form ${NAME} or ${NAME:-default}.
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-([^}]*))?\}`)

// expandEnv replaces every ${VAR} and ${VAR:-default} in the text with the
// environment variable's value. If the variable is not set and has no default
// either, it leaves an empty string.
func expandEnv(input string) string {
	return envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		groups := envVarPattern.FindStringSubmatch(match)
		name := groups[1]
		defaultVal := groups[3]
		if val, ok := os.LookupEnv(name); ok && val != "" {
			return val
		}
		return defaultVal
	})
}

// LoadConfig reads the YAML file at the given path, substitutes the environment
// variables and finally parses it.
func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	expanded := expandEnv(string(raw))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if len(c.Channels) == 0 {
		return fmt.Errorf("no channel is defined in config.yaml")
	}
	for _, rule := range c.Routes {
		for _, ch := range rule.Channels {
			if _, ok := c.Channels[ch]; !ok {
				return fmt.Errorf("rule %q references undefined channel %q", rule.Name, ch)
			}
		}
	}
	for _, ch := range c.DefaultChannels {
		if _, ok := c.Channels[ch]; !ok {
			return fmt.Errorf("default_channels references undefined channel %q", ch)
		}
	}
	return nil
}
