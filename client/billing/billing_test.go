package billing_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/threatwinds/go-sdk/client"
	"github.com/threatwinds/go-sdk/client/billing"
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

// setup creates a root client with a mock transport and returns the billing client.
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

func TestGetCustomer(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/customer" {
			t.Errorf("path = %q, want /customer", req.URL.Path)
		}

		return mockResp(200, nil, `{
			"id":"cust-123",
			"gcid":"gcid-abc"
		}`), nil
	}

	resp, err := c.Billing().GetCustomer(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "cust-123" {
		t.Errorf("ID = %q, want cust-123", resp.ID)
	}
	if resp.GCID != "gcid-abc" {
		t.Errorf("GCID = %q, want gcid-abc", resp.GCID)
	}
}

func TestAddMember(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "POST" {
			t.Errorf("method = %q, want POST", req.Method)
		}
		if req.URL.Path != "/customer/member" {
			t.Errorf("path = %q, want /customer/member", req.URL.Path)
		}

		var body billing.AddMemberRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body.UserID != "user-456" {
			t.Errorf("userID = %q, want user-456", body.UserID)
		}
		if body.Role != "editor" {
			t.Errorf("role = %q, want editor", body.Role)
		}

		return mockResp(204, nil, ""), nil
	}

	err := c.Billing().AddMember(context.Background(), billing.AddMemberRequest{
		UserID: "user-456",
		Role:   "editor",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListMembers(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/customer/members" {
			t.Errorf("path = %q, want /customer/members", req.URL.Path)
		}

		return mockResp(200, nil, `{
			"pages": 3,
			"items": 28,
			"members": [
				{"userID":"u1","role":"admin","email":"a@x.com","joinedAt":"2025-01-15T10:00:00Z"},
				{"userID":"u2","role":"viewer","email":"b@x.com","joinedAt":"2025-03-20T14:30:00Z"}
			]
		}`), nil
	}

	resp, err := c.Billing().ListMembers(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Pages != 3 {
		t.Errorf("Pages = %d, want 3", resp.Pages)
	}
	if resp.Items != 28 {
		t.Errorf("Items = %d, want 28", resp.Items)
	}
	if len(resp.Members) != 2 {
		t.Fatalf("len(Members) = %d, want 2", len(resp.Members))
	}
	if resp.Members[0].UserID != "u1" {
		t.Errorf("Members[0].UserID = %q, want u1", resp.Members[0].UserID)
	}
	if resp.Members[0].Role != "admin" {
		t.Errorf("Members[0].Role = %q, want admin", resp.Members[0].Role)
	}
	if resp.Members[1].Email != "b@x.com" {
		t.Errorf("Members[1].Email = %q, want b@x.com", resp.Members[1].Email)
	}
	expectedTime := time.Date(2025, 3, 20, 14, 30, 0, 0, time.UTC)
	if !resp.Members[1].JoinedAt.Equal(expectedTime) {
		t.Errorf("Members[1].JoinedAt = %v, want %v", resp.Members[1].JoinedAt, expectedTime)
	}
}

func TestAdminListCustomers(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/admin/customers" {
			t.Errorf("path = %q, want /admin/customers", req.URL.Path)
		}

		q := req.URL.Query()
		if q.Get("tierName") != "pro" {
			t.Errorf("tierName = %q, want pro", q.Get("tierName"))
		}
		if q.Get("status") != "active" {
			t.Errorf("status = %q, want active", q.Get("status"))
		}
		if q.Get("limit") != "20" {
			t.Errorf("limit = %q, want 20", q.Get("limit"))
		}

		return mockResp(200, nil, `{
			"pages": 2,
			"items": 35,
			"customers": [
				{"id":"c1","gcid":"g1","name":"Acme","email":"ops@acme.com","tierName":"pro","subscriptionStatus":"active","createdAt":"2025-01-01T00:00:00Z"}
			]
		}`), nil
	}

	resp, err := c.Billing().AdminListCustomers(context.Background(), &billing.AdminListCustomersOptions{
		Limit:    20,
		TierName: "pro",
		Status:   "active",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Pages != 2 {
		t.Errorf("Pages = %d, want 2", resp.Pages)
	}
	if resp.Items != 35 {
		t.Errorf("Items = %d, want 35", resp.Items)
	}
	if len(resp.Customers) != 1 {
		t.Fatalf("len(Customers) = %d, want 1", len(resp.Customers))
	}
	if resp.Customers[0].Name != "Acme" {
		t.Errorf("Customers[0].Name = %q, want Acme", resp.Customers[0].Name)
	}
	if resp.Customers[0].TierName != "pro" {
		t.Errorf("Customers[0].TierName = %q, want pro", resp.Customers[0].TierName)
	}
}

func TestGetAllLimits(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/limits" {
			t.Errorf("path = %q, want /limits", req.URL.Path)
		}

		return mockResp(200, nil, `{
			"customerId": "cust-123",
			"tierName": "pro",
			"limits": {
				"network": {
					"maxBandwidth": {
						"value": 1000,
						"window": "month",
						"description": "Maximum monthly bandwidth"
					}
				}
			}
		}`), nil
	}

	resp, err := c.Billing().GetAllLimits(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CustomerID != "cust-123" {
		t.Errorf("CustomerID = %q, want cust-123", resp.CustomerID)
	}
	if resp.TierName != "pro" {
		t.Errorf("TierName = %q, want pro", resp.TierName)
	}
	if resp.Limits == nil {
		t.Fatal("Limits is nil, want map")
	}
	if _, ok := resp.Limits["network"]; !ok {
		t.Error("missing 'network' key in Limits")
	}
}

func TestStartPortal(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Errorf("method = %q, want GET", req.Method)
		}
		if req.URL.Path != "/stripe/customer" {
			t.Errorf("path = %q, want /stripe/customer", req.URL.Path)
		}

		return mockResp(200, nil, `{
			"url": "https://billing.stripe.com/portal/session/apcs_test_abc123"
		}`), nil
	}

	resp, err := c.Billing().StartPortal(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedURL := "https://billing.stripe.com/portal/session/apcs_test_abc123"
	if resp.URL != expectedURL {
		t.Errorf("URL = %q, want %s", resp.URL, expectedURL)
	}
}

func TestAPIError_403(t *testing.T) {
	c, m := setup(t)

	m.roundTripper = func(req *http.Request) (*http.Response, error) {
		h := make(http.Header)
		h.Set("X-Error", "forbidden: insufficient permissions")
		h.Set("X-Error-Id", "billing-403")
		return mockResp(403, h, `{"error":"forbidden"}`), nil
	}

	_, err := c.Billing().GetCustomer(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*client.APIError)
	if !ok {
		t.Fatalf("expected *client.APIError, got %T", err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
	if apiErr.Message != "forbidden: insufficient permissions" {
		t.Errorf("Message = %q, want 'forbidden: insufficient permissions'", apiErr.Message)
	}
	if apiErr.ErrorID != "billing-403" {
		t.Errorf("ErrorID = %q, want 'billing-403'", apiErr.ErrorID)
	}
	if !apiErr.IsForbidden() {
		t.Error("IsForbidden() = false, want true")
	}
}
