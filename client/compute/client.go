package compute

// rootClient describes the methods a root SDK client must provide
// for the Compute service to function.
type rootClient interface{}

// Client provides access to the Compute API endpoints.
type Client struct {
	root rootClient
}

// NewClient creates a new Compute client backed by the root SDK client.
func NewClient(root rootClient) *Client {
	return &Client{root: root}
}
