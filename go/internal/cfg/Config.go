package cfg

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

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

type Socks5Input struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

type DirectOutput struct {
}

type Rate struct {
	Value    uint32
	Interval time.Duration
}

func (r *Rate) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	err := r.parse(s)
	if err != nil {
		return err
	}

	return nil
}

func (r *Rate) parse(s string) error {
	s = strings.TrimSpace(s)

	if s == "" {
		return fmt.Errorf("bad rate format %s", s)
	}

	if !strings.HasSuffix(s, "ps") {
		return fmt.Errorf("bad rate format %s", s)
	}
	s = strings.TrimSuffix(s, "ps")

	byteNames := []string{"b", "kb", "mb", "gb"}
	hasByteName := false
	i := len(byteNames) - 1
	for ; i >= 0; i-- {
		if strings.HasSuffix(s, byteNames[i]) {
			s = strings.TrimSuffix(s, byteNames[i])
			hasByteName = true
			break
		}
	}
	if !hasByteName {
		return fmt.Errorf("bad rate format %s", s)
	}

	value, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("bad rate format %s", s)
	}

	r.Value = uint32(value * int(math.Pow(1024, float64(i))))
	r.Interval = time.Second

	return nil
}

// TODO изучить встроенные структуры и их парсинг
// type Transport struct {
// Host      string `yaml:"host"`
// Port      uint16 `yaml:"port"`
// RateLimit Rate   `yaml:"rateLimit"`
// }

type TcpClientTransport struct {
	Host      string `yaml:"host"`
	Port      uint16 `yaml:"port"`
	RateLimit Rate   `yaml:"rateLimit"`
}

type TcpServerTransport struct {
	Host      string `yaml:"host"`
	Port      uint16 `yaml:"port"`
	RateLimit Rate   `yaml:"rateLimit"`
}

type YandexWebRTCTransport struct {
	JoinId    string `yaml:"joinId"`
	MailUser  string `yaml:"user"`
	MailPass  string `yaml:"pass"`
	RateLimit Rate   `yaml:"rateLimit"`
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
		var input Socks5Input
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
		var output DirectOutput
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
		var transport TcpClientTransport
		err = node.Decode(&transport)
		if err != nil {
			return err
		}
		c.Transport = &transport
	case "tcp-server":
		var transport TcpServerTransport
		err = node.Decode(&transport)
		if err != nil {
			return err
		}
		c.Transport = &transport
	case "ya-webrtc":
		var transport YandexWebRTCTransport
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
