package utils

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

func GZipEncode(buf []byte) ([]byte, error) {
	var compressedBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedBuf)

	_, err := gzipWriter.Write(buf)
	if err != nil {
		return nil, fmt.Errorf("gzip write error: %w", err)
	}

	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("gzip close error: %w", err)
	}

	return compressedBuf.Bytes(), nil
}

func GZipDecode(compressedBuf []byte) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedBuf))
	if err != nil {
		return nil, fmt.Errorf("gzip reader error: %w", err)
	}
	defer gzipReader.Close()

	buf, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("gzip read error: %w", err)
	}

	return buf, nil
}
