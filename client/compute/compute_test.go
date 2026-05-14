package compute_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/threatwinds/go-sdk/client"
	"github.com/threatwinds/go-sdk/client/compute"
)

// ---------------------------------------------------------------------------
// mockRT implements http.RoundTripper for unit tests.
// ---------------------------------------------------------------------------

type mockRT struct {
	roundTripper func(req *http.Request) (*http.Response, error)
}

func (m *mockRT) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.roundTripper != nil {
		return m.roundTripper(req)
	}
	return nil, nil
}

func mockResp(status int, headers http.Header, body string) *http.Response {
	if headers == nil {
		headers = make(http.Header)
	}
	return &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// setup creates a root client with a mock transport and returns the compute client.
func setup(t *testing.T) (*client.Client, *mockRT) {
	t.Helper()
	m := &mockRT{}
	c, err := client.New(
		client.WithAPIKey("k", "s"),
		client.WithHTTPClient(&http.Client{Transport: m}),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return c, m
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreateInstance(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "POST" {
			t.Errorf("method = %q, want POST", req.Method)
		}
		if req.URL.Path != "/instances" {
			t.Errorf("path = %q, want /instances", req.URL.Path)
		}

		var body compute.InstanceCreateRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body.TemplateID != "tpl-123" {
			t.Errorf("templateID = %q, want tpl-123", body.TemplateID)
		}
		if body.Zone != "us-east-1" {
			t.Errorf("zone = %q, want us-east-1", body.Zone)
		}

		return mockResp(200, nil, `{
			"id":"inst-abc",
			"userID":"user-1",
			"customerID":"cust-1",
			"name":"test-instance",
			"zone":"us-east-1",
			"machineType":"standard-2",
			"externalIp":"203.0.113.10",
			"internalIp":"10.0.0.5",
			"status":"running",
			"templateId":"tpl-123",
			"createdAt":"2026-05-14T00:00:00Z"
		}`), nil
	}

	resp, err := c.Compute().CreateInstance(context.Background(), compute.InstanceCreateRequest{
		TemplateID: "tpl-123",
		Zone:       "us-east-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "inst-abc" {
		t.Errorf("ID = %q, want inst-abc", resp.ID)
	}
	if resp.Name != "test-instance" {
		t.Errorf("Name = %q, want test-instance", resp.Name)
	}
	if resp.Status != "running" {
		t.Errorf("Status = %q, want running", resp.Status)
	}
	if resp.Zone != "us-east-1" {
		t.Errorf("Zone = %q, want us-east-1", resp.Zone)
	}
	if resp.ExternalIP != "203.0.113.10" {
		t.Errorf("ExternalIP = %q, want 203.0.113.10", resp.ExternalIP)
	}
}

func TestListInstances(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/instances" {
			t.Errorf("path = %q, want /instances", req.URL.Path)
		}

		return mockResp(200, nil, `[
			{"id":"inst-1","name":"web-1","status":"running","zone":"us-east-1"},
			{"id":"inst-2","name":"db-1","status":"stopped","zone":"eu-west-1"}
		]`), nil
	}

	instances, err := c.Compute().ListInstances(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instances) != 2 {
		t.Fatalf("len(instances) = %d, want 2", len(instances))
	}
	if instances[0].ID != "inst-1" {
		t.Errorf("instances[0].ID = %q, want inst-1", instances[0].ID)
	}
	if instances[0].Status != "running" {
		t.Errorf("instances[0].Status = %q, want running", instances[0].Status)
	}
	if instances[1].Status != "stopped" {
		t.Errorf("instances[1].Status = %q, want stopped", instances[1].Status)
	}
}

func TestGetInstance(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/instances/inst-abc" {
			t.Errorf("path = %q, want /instances/inst-abc", req.URL.Path)
		}

		return mockResp(200, nil, `{
			"id":"inst-abc",
			"name":"web-1",
			"status":"running",
			"zone":"us-east-1",
			"machineType":"standard-4"
		}`), nil
	}

	resp, err := c.Compute().GetInstance(context.Background(), "inst-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "inst-abc" {
		t.Errorf("ID = %q, want inst-abc", resp.ID)
	}
	if resp.MachineType != "standard-4" {
		t.Errorf("MachineType = %q, want standard-4", resp.MachineType)
	}
}

func TestStartInstance(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "POST" {
			t.Errorf("method = %q, want POST", req.Method)
		}
		if req.URL.Path != "/instances/inst-abc/start" {
			t.Errorf("path = %q, want /instances/inst-abc/start", req.URL.Path)
		}

		return mockResp(204, nil, ""), nil
	}

	err := c.Compute().StartInstance(context.Background(), "inst-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListTemplates(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/templates" {
			t.Errorf("path = %q, want /templates", req.URL.Path)
		}

		return mockResp(200, nil, `[
			{"id":"tpl-1","name":"Ubuntu 24.04","machineType":"standard-2","diskSizeGb":50,"diskType":"ssd","image":"ubuntu-24.04","region":"us-east"},
			{"id":"tpl-2","name":"Debian 12","machineType":"standard-4","diskSizeGb":100,"diskType":"nvme","image":"debian-12","region":"eu-west"}
		]`), nil
	}

	templates, err := c.Compute().ListTemplates(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("len(templates) = %d, want 2", len(templates))
	}
	if templates[0].ID != "tpl-1" {
		t.Errorf("templates[0].ID = %q, want tpl-1", templates[0].ID)
	}
	if templates[0].DiskSizeGb != 50 {
		t.Errorf("templates[0].DiskSizeGb = %d, want 50", templates[0].DiskSizeGb)
	}
	if templates[1].DiskType != "nvme" {
		t.Errorf("templates[1].DiskType = %q, want nvme", templates[1].DiskType)
	}
}

func TestAdminListInstances(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/admin/instances" {
			t.Errorf("path = %q, want /admin/instances", req.URL.Path)
		}

		q := req.URL.Query()
		if q.Get("limit") != "10" {
			t.Errorf("limit = %q, want 10", q.Get("limit"))
		}
		if q.Get("page") != "1" {
			t.Errorf("page = %q, want 1", q.Get("page"))
		}
		if q.Get("status") != "running" {
			t.Errorf("status = %q, want running", q.Get("status"))
		}
		if q.Get("zone") != "us-east-1" {
			t.Errorf("zone = %q, want us-east-1", q.Get("zone"))
		}

		return mockResp(200, nil, `{
			"pages":3,
			"items":25,
			"instances":[
				{"id":"inst-1","userID":"user-1","name":"web-1","status":"running","zone":"us-east-1"},
				{"id":"inst-2","userID":"user-2","name":"db-1","status":"running","zone":"us-east-1"}
			]
		}`), nil
	}

	resp, err := c.Compute().AdminListInstances(context.Background(), &compute.AdminListInstancesOptions{
		Limit:  10,
		Page:   1,
		Status: "running",
		Zone:   "us-east-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Pages != 3 {
		t.Errorf("Pages = %d, want 3", resp.Pages)
	}
	if resp.Items != 25 {
		t.Errorf("Items = %d, want 25", resp.Items)
	}
	if len(resp.Instances) != 2 {
		t.Fatalf("len(Instances) = %d, want 2", len(resp.Instances))
	}
	if resp.Instances[0].ID != "inst-1" {
		t.Errorf("Instances[0].ID = %q, want inst-1", resp.Instances[0].ID)
	}
	if resp.Instances[1].UserID != "user-2" {
		t.Errorf("Instances[1].UserID = %q, want user-2", resp.Instances[1].UserID)
	}
}

func TestAPIError_404(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		h := make(http.Header)
		h.Set("X-Error", "instance not found")
		h.Set("X-Error-Id", "compute-404")
		return mockResp(404, h, `{"error":"not found"}`), nil
	}

	_, err := c.Compute().GetInstance(context.Background(), "inst-missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*client.APIError)
	if !ok {
		t.Fatalf("expected *client.APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Message != "instance not found" {
		t.Errorf("Message = %q, want 'instance not found'", apiErr.Message)
	}
	if apiErr.ErrorID != "compute-404" {
		t.Errorf("ErrorID = %q, want 'compute-404'", apiErr.ErrorID)
	}
	if !apiErr.IsNotFound() {
		t.Error("IsNotFound() = false, want true")
	}
}
