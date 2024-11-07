package go_sdk

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/threatwinds/logger"
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
//   - *logger.Error: An error if any occurred during the request or response
//     processing, otherwise nil.
func DoReq[response any](url string,
	data []byte, method string,
	headers map[string]string) (response, int, *logger.Error) {

	var result response

	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return result,
			http.StatusInternalServerError,
			Logger().ErrorF("error creating request: %s", err.Error())
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return result,
			http.StatusInternalServerError,
			Logger().ErrorF("error sending request: %s", err.Error())
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result,
			http.StatusInternalServerError,
			Logger().ErrorF("error reading response: %s", err.Error())
	}

	if resp.StatusCode >= 400 {
		return result,
			resp.StatusCode,
			Logger().ErrorF("received status code %d", resp.StatusCode)
	}

	if resp.StatusCode == http.StatusNoContent{
		return result, resp.StatusCode, nil
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return result,
			http.StatusInternalServerError,
			Logger().ErrorF("error unmarshalling response: %s", err.Error())
	}

	return result, resp.StatusCode, nil
}

// Download downloads the content from the specified URL and saves it to the specified file.
// It returns a *logger.Error if any error occurs during the process.
//
// Parameters:
//   - url: The URL from which to download the content.
//   - file: The path to the file where the content should be saved.
//
// Returns:
//   - *logger.Error: An error object if an error occurs, otherwise nil.
func Download(url, file string) *logger.Error {
	out, err := os.Create(file)
	if err != nil {
		return Logger().ErrorF("could not create file: %s", err.Error())
	}

	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return Logger().ErrorF("could not do request to the URL: %s", err.Error())
	}

	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return Logger().ErrorF("could not save data to file: %s", err.Error())
	}

	return nil
}
