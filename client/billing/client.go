package billing

// rootClient describes the methods a root SDK client must provide
// for the Billing service to function.
type rootClient interface{}

// Client provides access to the Billing API endpoints.
type Client struct {
	root rootClient
}

// NewClient creates a new Billing client backed by the root SDK client.
func NewClient(root rootClient) *Client {
	return &Client{root: root}
}
