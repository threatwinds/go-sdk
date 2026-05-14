package auth

// rootClient describes the methods a root SDK client must provide
// for the Auth service to function.
type rootClient interface{}

// Client provides access to the Auth API endpoints.
type Client struct {
	root rootClient
}

// NewClient creates a new Auth client backed by the root SDK client.
func NewClient(root rootClient) *Client {
	return &Client{root: root}
}
