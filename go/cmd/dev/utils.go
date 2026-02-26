package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

func request(proxyAddress, url string) string {
	start := time.Now()

	dialer, err := proxy.SOCKS5("tcp", proxyAddress, nil, proxy.Direct)
	if err != nil {
		panic(err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			Dial: dialer.Dial,
		},
	}

	slog.Debug(fmt.Sprintf("[dev request] via SOCKS5 proxy [%s] to %s", proxyAddress, url))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("User-Agent", "curl")

	resp, err := httpClient.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	bodyStr := string(body)

	elapsed := time.Since(start)

	slog.Debug(fmt.Sprintf("Request via SOCKS5 proxy [%s] to %s done with status %s and body %s, took %s", proxyAddress, url, resp.Status, bodyStr, elapsed))

	return bodyStr
}
