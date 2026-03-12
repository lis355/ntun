package conf

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

type Config struct {
	Name      string      `yaml:"name"`
	Id        uuid.UUID   `yaml:"id"`
	Allowed   []uuid.UUID `yaml:"allowed"`
	CipherKey string      `yaml:"pass"`
	Input     any         `yaml:"-"`
	Output    any         `yaml:"-"`
	Transport any         `yaml:"-"`
}

type InputSocks5 struct {
	Port uint16 `yaml:"port"`
}

type OutputDirect struct {
}

type TransportTcpClient struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

type TransportTcpServer struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

func (c *Config) ParseFile(filePath string) error {
	buf, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	return c.Parse(buf)
}

func (c *Config) Parse(buf []byte) error {
	err := yaml.Unmarshal(buf, &c)
	if err != nil {
		return err
	}

	var raw struct {
		Input     yaml.Node `yaml:"input"`
		Output    yaml.Node `yaml:"output"`
		Transport yaml.Node `yaml:"transport"`
	}

	err = yaml.Unmarshal(buf, &raw)

	c.parseInput(raw.Input)
	c.parseOutput(raw.Output)
	c.parseTransport(raw.Transport)

	return nil
}

func (c *Config) parseInput(node yaml.Node) error {
	if node.IsZero() {
		return nil
	}

	var raw struct {
		Typ string `yaml:"type"`
	}

	err := node.Decode(&raw)
	if err != nil {
		return err
	}

	switch raw.Typ {
	case "socks5":
		var input InputSocks5
		err = node.Decode(&input)
		if err != nil {
			return err
		}
		c.Input = &input
	default:
		return fmt.Errorf("Bad input type %s", raw.Typ)
	}

	return nil
}

func (c *Config) parseOutput(node yaml.Node) error {
	if node.IsZero() {
		return nil
	}

	var raw struct {
		Typ string `yaml:"type"`
	}

	err := node.Decode(&raw)
	if err != nil {
		return err
	}

	switch raw.Typ {
	case "direct":
		var output OutputDirect
		c.Output = &output
	default:
		return fmt.Errorf("Bad output type %s", raw.Typ)
	}

	return nil
}

func (c *Config) parseTransport(node yaml.Node) error {
	if node.IsZero() {
		return nil
	}

	var raw struct {
		Typ string `yaml:"type"`
	}

	err := node.Decode(&raw)
	if err != nil {
		return err
	}

	switch raw.Typ {
	case "tcp-client":
		var transport TransportTcpClient
		err = node.Decode(&transport)
		if err != nil {
			return err
		}
		c.Transport = &transport
	case "tcp-server":
		var transport TransportTcpServer
		err = node.Decode(&transport)
		if err != nil {
			return err
		}
		c.Transport = &transport
	default:
		return fmt.Errorf("Bad transport type %s", raw.Typ)
	}

	return nil
}
