package helpers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/threatwinds/logger"
)

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
