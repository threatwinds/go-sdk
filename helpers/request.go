package helpers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/threatwinds/logger"
)

func DoReq[response any](url string, data []byte, method string, headers map[string]string) (response, int, *logger.Error) {
	var result response

	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return result, http.StatusInternalServerError, Logger().ErrorF("error creating request: %s", err.Error())
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return result, http.StatusInternalServerError, Logger().ErrorF("error sending request: %s", err.Error())
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, http.StatusInternalServerError, Logger().ErrorF("error reading response: %s", err.Error())
	}

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return result, resp.StatusCode, Logger().ErrorF("received status code %d", resp.StatusCode)
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return result, http.StatusInternalServerError, Logger().ErrorF("error unmarshalling response: %s", err.Error())
	}

	return result, resp.StatusCode, nil
}
