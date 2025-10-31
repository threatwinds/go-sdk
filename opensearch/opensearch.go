package opensearch

import (
	"crypto/tls"
	"github.com/threatwinds/go-sdk/catcher"
	"net/http"
	"sync"

	osgo "github.com/opensearch-project/opensearch-go/v2"
)

var client *osgo.Client

var once = sync.Once{}

func Connect(nodes []string) error {
	var err error

	once.Do(func() {
		client, err = osgo.NewClient(osgo.Config{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Addresses: nodes,
		})
	})

	if err != nil {
		return catcher.Error("cannot connect to OpenSearch", err, nil)
	}

	return nil
}
