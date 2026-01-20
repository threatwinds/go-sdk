package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// DoReq sends an HTTP request and processes the response.
//
// This function sends an HTTP request to the specified URL with the given
// method, data, and headers. It returns the response body unmarshalled into
// the specified response type, the HTTP status code, and an error if any
// occurred during the process.
//
// Type Parameters:
//   - response: The type into which the response body will be unmarshalled.
//
// Parameters:
//   - url: The URL to which the request is sent.
//   - data: The request payload as a byte slice.
//   - method: The HTTP method to use for the request (e.g., "GET", "POST").
//   - headers: A map of headers to include in the request.
//
// Returns:
//   - response: The response body unmarshalled into the specified type.
//   - int: The HTTP status code of the response.
//   - error: An error if any occurred during the request or response
//     processing, otherwise nil.
func DoReq[response any](url string, data []byte, method string, headers map[string]string) (response, int, error) {
	var result response

	if len(data) > maxMessageSize {
		return result, http.StatusBadRequest, fmt.Errorf("cannot convert to object: data size exceeds limit (size=%d bytes, limit=%d bytes)", len(data), maxMessageSize)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("error creating request: %w", err)
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	// Configure HTTP client with security settings
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			DisableCompression: true,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("error doing request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, http.StatusInternalServerError, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return result, resp.StatusCode, fmt.Errorf("error response (status=%d): %s", resp.StatusCode, string(body))
	}

	if resp.StatusCode == http.StatusNoContent {
		return result, resp.StatusCode, nil
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return result, resp.StatusCode, fmt.Errorf("error parsing response: %w", err)
	}

	return result, resp.StatusCode, nil
}

// DownloadOption defines a functional option for configuring the download request.
type DownloadOption func(*downloadConfig)

type downloadConfig struct {
	headers map[string]string
	timeout time.Duration
}

// WithHeaders sets the headers for the download request.
func WithHeaders(headers map[string]string) DownloadOption {
	return func(c *downloadConfig) {
		c.headers = headers
	}
}

// WithTimeout sets the timeout for the download request.
func WithTimeout(timeout time.Duration) DownloadOption {
	return func(c *downloadConfig) {
		c.timeout = timeout
	}
}

// Download downloads the content from the specified URL and saves it to the specified file.
// It returns an error if any error occurs during the process.
//
// Parameters:
//   - url: The URL from which to download the content.
//   - file: The path to the file where the content should be saved.
//   - opts: Optional functional options to configure the download (e.g., WithHeaders, WithTimeout).
//
// Returns:
//   - error: An error object if an error occurs, otherwise nil.
func Download(url, file string, opts ...DownloadOption) error {
	config := &downloadConfig{
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(config)
	}

	out, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", file, err)
	}
	defer func() { _ = out.Close() }()

	resp, err := DownloadStream(url, opts...)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Close() }()

	_, err = io.Copy(out, resp)
	if err != nil {
		return fmt.Errorf("error saving file %s: %w", file, err)
	}

	return nil
}

// DownloadStream downloads the content from the specified URL and returns it as an io.ReadCloser.
// The caller is responsible for closing the returned stream.
//
// Parameters:
//   - url: The URL from which to download the content.
//   - opts: Optional functional options to configure the download (e.g., WithHeaders, WithTimeout).
//
// Returns:
//   - io.ReadCloser: The response body as a stream.
//   - error: An error object if an error occurs, otherwise nil.
func DownloadStream(url string, opts ...DownloadOption) (io.ReadCloser, error) {
	config := &downloadConfig{
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(config)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	for k, v := range config.headers {
		req.Header.Add(k, v)
	}

	client := &http.Client{
		Timeout: config.timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error downloading from %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("bad status downloading from %s: %s", url, resp.Status)
	}

	return resp.Body, nil
}
