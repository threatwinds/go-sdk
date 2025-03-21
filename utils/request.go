package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/threatwinds/go-sdk/catcher"
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
		return result, http.StatusBadRequest, catcher.Error("cannot convert to object",
			errors.New("data size exceeds limit"), map[string]any{
				"size":  fmt.Sprintf("%d bytes", len(data)),
				"limit": fmt.Sprintf("%d bytes", maxMessageSize),
			})
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return result, http.StatusInternalServerError, catcher.Error("error creating request", err, nil)
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
		return result, http.StatusInternalServerError, catcher.Error("error doing request", err, nil)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, http.StatusInternalServerError, catcher.Error("error reading response body", err, nil)
	}

	if resp.StatusCode >= 400 {
		return result, resp.StatusCode, catcher.Error("error response", nil, map[string]interface{}{
			"response": string(body),
			"status":   resp.StatusCode,
		})
	}

	if resp.StatusCode == http.StatusNoContent {
		return result, resp.StatusCode, nil
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return result, resp.StatusCode, catcher.Error("error parsing response", err, nil)
	}

	return result, resp.StatusCode, nil
}

// Download downloads the content from the specified URL and saves it to the specified file.
// It returns an error if any error occurs during the process.
//
// Parameters:
//   - url: The URL from which to download the content.
//   - file: The path to the file where the content should be saved.
//
// Returns:
//   - error: An error object if an error occurs, otherwise nil.
func Download(url, file string) error {
	out, err := os.Create(file)
	if err != nil {
		return catcher.Error("error creating file", err, map[string]interface{}{"file": file})
	}

	defer func() { _ = out.Close() }()

	// Add secure HTTP client configuration
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return catcher.Error("error downloading file", err, map[string]any{"url": url})
	}

	defer func() { _ = resp.Body.Close() }()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return catcher.Error("error saving file", err, map[string]any{"file": file})
	}

	return nil
}
