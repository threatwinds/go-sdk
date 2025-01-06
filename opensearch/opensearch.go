package opensearch

import (
	"crypto/tls"
	"github.com/threatwinds/go-sdk/catcher"
	"net/http"
	"sync"

	osgo "github.com/opensearch-project/opensearch-go/v2"
)

var (
	client *osgo.Client
	err    error
)

var once = sync.Once{}

func Connect(nodes []string) error {
	once.Do(func() {
		client, err = osgo.NewClient(osgo.Config{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Addresses: nodes,
		})
	})

	return catcher.Error("failed to connect to OpenSearch", err, map[string]any{"nodes": nodes})
}
