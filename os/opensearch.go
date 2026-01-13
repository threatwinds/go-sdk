package os

import (
	"crypto/tls"
	"net/http"
	"sync"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

var (
	client    *opensearch.Client
	apiClient *opensearchapi.Client
	err       error
)

var once = sync.Once{}

// Connect establishes a singleton connection to OpenSearch.
// Only the first successful call takes effect; later calls return the existing connection.
// The connection uses TLS with certificate verification disabled.
func Connect(nodes []string, user, password string) error {
	if apiClient != nil {
		return nil
	}

	once.Do(func() {
		apiClient, err = opensearchapi.NewClient(opensearchapi.Config{
			Client: opensearch.Config{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
				Addresses: nodes,
				Username:  user,
				Password:  password,
			},
		})
		if err == nil {
			client = apiClient.Client
		}
	})

	if err != nil {
		// Reset once to allow retry on next call if initial attempt failed
		once = sync.Once{}
	}

	return err
}
