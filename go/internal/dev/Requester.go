package dev

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

type Requester struct {
	socks5ProxyAddress string
	dialer             proxy.Dialer
}

func NewRequester(socks5ProxyAddress string) (*Requester, error) {
	dialer, err := proxy.SOCKS5("tcp", socks5ProxyAddress, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}

	return &Requester{
		socks5ProxyAddress: socks5ProxyAddress,
		dialer:             dialer,
	}, nil
}

func (r *Requester) Get(url string) (string, error) {
	start := time.Now()

	httpClient := &http.Client{
		Transport: &http.Transport{
			Dial:              r.dialer.Dial,
			DisableKeepAlives: true,
		},
	}

	slog.Debug(fmt.Sprintf("[Requester.Get] [socks5://%s] --> %s", r.socks5ProxyAddress, url))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "curl")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	bodyStr := string(body)

	elapsed := time.Since(start)

	slog.Debug(fmt.Sprintf("[Requester.Get] %s [socks5://%s] %v [%s] %s", url, r.socks5ProxyAddress, elapsed, resp.Status, bodyStr))

	return bodyStr, nil
}

func (r *Requester) Post(url, requestBody string) (string, error) {
	start := time.Now()

	httpClient := &http.Client{
		Transport: &http.Transport{
			Dial:              r.dialer.Dial,
			DisableKeepAlives: true,
		},
	}

	slog.Debug(fmt.Sprintf("[Requester.Post] [socks5://%s] --> %s %s", r.socks5ProxyAddress, url, requestBody))

	var bodyReader io.Reader
	if requestBody != "" {
		bodyReader = strings.NewReader(requestBody)
	}

	req, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "curl")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	bodyStr := string(body)

	elapsed := time.Since(start)

	slog.Debug(fmt.Sprintf("[Requester.Post] %s [socks5://%s] %v [%s] %s", url, r.socks5ProxyAddress, elapsed, resp.Status, bodyStr))

	return bodyStr, nil
}
