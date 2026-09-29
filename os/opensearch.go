package os

import (
	"crypto/tls"
	"net/http"
	"sync"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
)

// Error ids in this package, stated once here because every request below goes
// through the client this file builds.
//
// OpenSearch is a third party. It does not run catcher.GinError and therefore
// never sends x-error-id, so there is no upstream occurrence id to adopt — and
// no error response header is read anywhere in this package, deliberately.
// That is the whole of the difference from utils and client, which do talk to
// ThreatWinds peers and do adopt. A header from OpenSearch that happened to be
// spelled x-error-id would not be one of this org's ids and must not be treated
// as one.
//
// Failures here are still traceable, because every one of them is identified
// exactly once by catcher at the point it becomes an *SdkError: at the
// catcher.Error call inside this package for the paths that build one, and at
// the caller's wrap for the paths that return a plain error. Generating an id
// at the request itself — as utils does for its transport failures — would buy
// nothing here and cost a second id per failure. The reason it is worth doing
// in utils is that a transport error there may be reported by a caller that
// never involves catcher at all, and that the failure crosses a service
// boundary an operator has to follow; neither is true of a query against this
// process's own datastore.

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
