package compute

// AdminListInstancesOptions provides filtering for the admin instance list endpoint.
type AdminListInstancesOptions struct {
	Limit      int    `url:"limit,omitempty"`
	Page       int    `url:"page,omitempty"`
	UserID     string `url:"userID,omitempty"`
	CustomerID string `url:"customerID,omitempty"`
	Status     string `url:"status,omitempty"`
	Zone       string `url:"zone,omitempty"`
	TemplateID string `url:"templateID,omitempty"`
}

// ListOpts holds pagination parameters for admin list endpoints.
type ListOpts struct {
	Limit int
	Page  int
}

// — Instance —

// InstanceCreateRequest is the body for creating a new instance.
type InstanceCreateRequest struct {
	TemplateID string `json:"templateID"`
	Zone       string `json:"zone"`
}

// Instance represents a compute instance.
type Instance struct {
	ID          string `json:"id"`
	UserID      string `json:"userID"`
	CustomerID  string `json:"customerID"`
	Name        string `json:"name"`
	Zone        string `json:"zone"`
	MachineType string `json:"machineType"`
	ExternalIP  string `json:"externalIp"`
	InternalIP  string `json:"internalIp"`
	Status      string `json:"status"`
	TemplateID  string `json:"templateId"`
	CreatedAt   string `json:"createdAt"`
}

// — Templates —

// Template represents a compute instance template.
type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MachineType string `json:"machineType"`
	DiskSizeGb  int    `json:"diskSizeGb"`
	DiskType    string `json:"diskType"`
	Image       string `json:"image"`
	Region      string `json:"region"`
}

// — Paginated Responses —

// AdminListInstancesResponse is the paginated response for admin instance listing.
type AdminListInstancesResponse struct {
	Pages     int        `json:"pages"`
	Items     int        `json:"items"`
	Instances []Instance `json:"instances"`
}
