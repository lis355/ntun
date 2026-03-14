package conf

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

func ParseUUID(str string) uuid.UUID {
	id, _ := uuid.Parse(str)

	return id
}

func TestConfig(t *testing.T) {
	var tests = []struct {
		name string
		in   string
		out  *Config
	}{
		{
			"client",
			`
		name: client
		id: 6a971624-0b19-4c8d-8ceb-d5544675e898
		allowed: 
			- 1ee18100-39f7-434c-9206-47bfe572169d
		pass: "123"
		input: 
			type: socks5
			port: 8080
		transport: 
			type: tcp-client
			host: localhost
			port: 8303
			rateLimit: 50mbps
		`, &Config{
				Name:      "client",
				Id:        ParseUUID("6a971624-0b19-4c8d-8ceb-d5544675e898"),
				Allowed:   []uuid.UUID{ParseUUID("1ee18100-39f7-434c-9206-47bfe572169d")},
				CipherKey: "123",
				Input: &InputSocks5{
					Port: 8080,
				},
				Transport: &TransportTcpClient{
					Host:      "localhost",
					Port:      8303,
					RateLimit: Rate{50 * 1024 * 1024},
				},
			},
		},
		{
			"server",
			`
		name: client
		id: 6a971624-0b19-4c8d-8ceb-d5544675e898
		allowed: 
			- 1ee18100-39f7-434c-9206-47bfe572169d
		pass: "123"
		output: 
			type: direct
		transport: 
			type: tcp-server
			host: localhost
			port: 8303
			rateLimit: 50mbps
		`, &Config{
				Name:      "client",
				Id:        ParseUUID("6a971624-0b19-4c8d-8ceb-d5544675e898"),
				Allowed:   []uuid.UUID{ParseUUID("1ee18100-39f7-434c-9206-47bfe572169d")},
				CipherKey: "123",
				Output:    &OutputDirect{},
				Transport: &TransportTcpServer{
					Host:      "localhost",
					Port:      8303,
					RateLimit: Rate{50 * 1024 * 1024},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			err := cfg.Parse([]byte(strings.ReplaceAll(tt.in, "\t", "  ")))

			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(&cfg, tt.out); diff != "" {
				t.Errorf("cfg, tt.out different (-want +got):\n%s", diff)
			}
		})
	}
}
