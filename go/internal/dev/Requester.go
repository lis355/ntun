package dev

import (
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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

func (r *Requester) Get(urlStr string) (string, error) {
	start := time.Now()

	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	insecureSkipVerify := false

	if u.Scheme == "https" &&
		strings.HasPrefix(u.Hostname(), "localhost") {
		insecureSkipVerify = true
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			Dial:              r.dialer.Dial,
			DisableKeepAlives: true,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: insecureSkipVerify},
		},
	}

	// slog.Debug(fmt.Sprintf("[Requester.Get] [socks5://%s] --> %s", r.socks5ProxyAddress, urlStr))

	req, err := http.NewRequest("GET", urlStr, nil)
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

	slog.Debug(fmt.Sprintf("[Requester.Get] [socks5://%s] --> %s %v [%s] %s", r.socks5ProxyAddress, urlStr, elapsed, resp.Status, bodyStr[:min(len(bodyStr), 10)]))

	return bodyStr, nil
}

func (r *Requester) Post(urlStr, requestBody string) (string, error) {
	start := time.Now()

	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	insecureSkipVerify := false

	if u.Scheme == "https" &&
		strings.HasPrefix(u.Hostname(), "localhost") {
		insecureSkipVerify = true
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			Dial:              r.dialer.Dial,
			DisableKeepAlives: true,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: insecureSkipVerify},
		},
	}

	// slog.Debug(fmt.Sprintf("[Requester.Post] [socks5://%s] --> %s %s", r.socks5ProxyAddress, urlStr, requestBody))

	var bodyReader io.Reader
	if requestBody != "" {
		bodyReader = strings.NewReader(requestBody)
	}

	req, err := http.NewRequest("POST", urlStr, bodyReader)
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

	slog.Debug(fmt.Sprintf("[Requester.Post] [socks5://%s] --> %s %v [%s] %s", r.socks5ProxyAddress, urlStr, elapsed, resp.Status, bodyStr[:min(len(bodyStr), 10)]))

	return bodyStr, nil
}
