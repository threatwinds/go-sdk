package go_sdk

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
func DoReq[response any](url string,
	data []byte, method string,
	headers map[string]string) (response, int, error) {

	var result response

	if len(data) > maxMessageSize {
		return result, http.StatusRequestEntityTooLarge, fmt.Errorf("request too large")
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return result,
			http.StatusInternalServerError,
			Error(Trace(), map[string]interface{}{
				"cause": err.Error(),
				"error": "error creating request",
			})
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
		return result,
			http.StatusInternalServerError,
			Error(Trace(), map[string]interface{}{
				"cause": err.Error(),
				"error": "error doing request",
			})
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result,
			http.StatusInternalServerError,
			Error(Trace(), map[string]interface{}{
				"cause": err.Error(),
				"error": "error reading response body",
			})
	}

	if resp.StatusCode >= 400 {
		return result,
			resp.StatusCode,
			Error(Trace(), map[string]interface{}{
				"error":  "error response",
				"status": resp.StatusCode,
			})
	}

	if resp.StatusCode == http.StatusNoContent {
		return result, resp.StatusCode, nil
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return result,
			resp.StatusCode,
			Error(Trace(), map[string]interface{}{
				"cause": err.Error(),
				"error": "error parsing response",
			})
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
		return Error(Trace(), map[string]interface{}{
			"cause": err.Error(),
			"error": "error creating file",
			"file":  file,
		})
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
		return Error(Trace(), map[string]interface{}{
			"cause": err.Error(),
			"error": "error downloading file",
			"url":   url,
		})
	}

	defer func() { _ = resp.Body.Close() }()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return Error(Trace(), map[string]interface{}{
			"cause": err.Error(),
			"error": "error saving file",
			"file":  file,
		})
	}

	return nil
}
