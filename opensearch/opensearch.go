package opensearch

import (
	"crypto/tls"
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

	return err
}
