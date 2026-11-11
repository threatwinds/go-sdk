package opensearch

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkos "github.com/threatwinds/go-sdk/os"
	"github.com/threatwinds/go-sdk/store"
)

// ismPolicy is the subset this driver reads. It is partial, which is why
// SetRetention refuses to write the document back.
type ismPolicy struct {
	Policy struct {
		PolicyID     string `json:"policy_id"`
		DefaultState string `json:"default_state"`
		States       []struct {
			Name        string `json:"name"`
			Transitions []struct {
				StateName  string `json:"state_name"`
				Conditions *struct {
					MinIndexAge string `json:"min_index_age"`
				} `json:"conditions"`
			} `json:"transitions"`
		} `json:"states"`
	} `json:"policy"`
}

func (d *Driver) readPolicy(ctx context.Context) (*ismPolicy, error) {
	id := d.cfg.RetentionPolicyID
	if id == "" {
		return nil, fmt.Errorf("%w: Retention (no RetentionPolicyID configured)", store.ErrUnsupported)
	}
	body, status, err := sdkos.DoRequest(ctx, "GET", "/_plugins/_ism/policies/"+id, nil)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("opensearch: reading ISM policy %s returned %d", id, status)
	}
	var p ismPolicy
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("opensearch: decoding ISM policy: %w", err)
	}
	return &p, nil
}

// policyRetention reads the age of whichever transition deletes, and the age of
// whichever moves an index off hot storage. The second is the tiering: on this
// engine a colder tier is a separate state the index transitions into.
func policyRetention(p *ismPolicy) (age, coldAge string) {
	for _, st := range p.Policy.States {
		for _, tr := range st.Transitions {
			minAge := ""
			if tr.Conditions != nil {
				minAge = tr.Conditions.MinIndexAge
			}
			switch tr.StateName {
			case "delete", "safe_delete":
				if minAge != "" {
					age = minAge
				}
			case "warm", "cold", "backup":
				if minAge != "" && coldAge == "" {
					coldAge = minAge
				}
			}
		}
	}
	return age, coldAge
}

// parseIndexAge exists because time.ParseDuration rejects the "d" unit that
// every retention setting here uses.
func parseIndexAge(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, nil
	}
	// Longest suffix first so "ms" is never read as "s".
	units := []struct {
		suffix string
		unit   time.Duration
	}{
		{"ms", time.Millisecond},
		{"d", 24 * time.Hour},
		{"h", time.Hour},
		{"m", time.Minute},
		{"s", time.Second},
	}
	for _, u := range units {
		if !strings.HasSuffix(s, u.suffix) {
			continue
		}
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, u.suffix), 64)
		if err != nil {
			continue
		}
		return time.Duration(n * float64(u.unit)), nil
	}
	return 0, fmt.Errorf("opensearch: cannot parse index age %q", s)
}

// SetRetention patches the deleting transition of the raw policy document and
// writes it back, so fields this package does not model survive the round trip.
func (d *Driver) SetRetention(ctx context.Context, dataset store.Dataset, r store.Retention) error {
	id := d.cfg.RetentionPolicyID
	if id == "" {
		return fmt.Errorf("%w: SetRetention (no RetentionPolicyID configured)", store.ErrUnsupported)
	}
	if r.Keep <= 0 {
		return fmt.Errorf("opensearch: SetRetention needs a positive Keep")
	}

	body, status, err := sdkos.DoRequest(ctx, "GET", "/_plugins/_ism/policies/"+id, nil)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("opensearch: reading ISM policy %s returned %d", id, status)
	}

	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("opensearch: decoding ISM policy: %w", err)
	}
	policy, ok := doc["policy"].(map[string]any)
	if !ok {
		return fmt.Errorf("opensearch: ISM policy %s has no policy object", id)
	}

	var current ismPolicy
	if err := json.Unmarshal(body, &current); err != nil {
		return fmt.Errorf("opensearch: decoding ISM policy: %w", err)
	}
	// Tiering here means adding or removing states in the policy document, which
	// is more than patching an age. Refusing keeps a caller from believing the
	// tier changed when only the deletion age did.
	if _, coldAge := policyRetention(&current); (coldAge != "") != r.Tiered() {
		return fmt.Errorf("%w: SetRetention cannot add or remove a cold tier, only change Keep", store.ErrUnsupported)
	}

	if n := patchDeleteAge(policy, formatIndexAge(r.Keep)); n == 0 {
		return fmt.Errorf("opensearch: ISM policy %s has no transition into a deleting state", id)
	}

	path := fmt.Sprintf("/_plugins/_ism/policies/%s?if_seq_no=%v&if_primary_term=%v",
		id, doc["_seq_no"], doc["_primary_term"])
	body, status, err = sdkos.DoRequest(ctx, "PUT", path, map[string]any{"policy": policy})
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("opensearch: writing ISM policy %s returned %d: %s", id, status, string(body))
	}
	return nil
}

// patchDeleteAge rewrites min_index_age on every transition into a deleting
// state and reports how many it changed.
func patchDeleteAge(policy map[string]any, age string) int {
	states, _ := policy["states"].([]any)
	patched := 0
	for _, s := range states {
		st, ok := s.(map[string]any)
		if !ok {
			continue
		}
		transitions, _ := st["transitions"].([]any)
		for _, t := range transitions {
			tr, ok := t.(map[string]any)
			if !ok {
				continue
			}
			name, _ := tr["state_name"].(string)
			if name != "delete" && name != "safe_delete" {
				continue
			}
			cond, ok := tr["conditions"].(map[string]any)
			if !ok {
				cond = map[string]any{}
				tr["conditions"] = cond
			}
			cond["min_index_age"] = age
			patched++
		}
	}
	return patched
}

// formatIndexAge is the inverse of parseIndexAge, preferring the coarsest unit
// that divides exactly.
func formatIndexAge(d time.Duration) string {
	switch {
	case d%(24*time.Hour) == 0:
		return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
	case d%time.Hour == 0:
		return fmt.Sprintf("%dh", int(d/time.Hour))
	case d%time.Minute == 0:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	default:
		return fmt.Sprintf("%ds", int(d/time.Second))
	}
}
