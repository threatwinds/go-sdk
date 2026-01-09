package os

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ISMPolicy represents an OpenSearch Index State Management policy
type ISMPolicy struct {
	PolicyID     string        `json:"policy_id,omitempty"`
	Description  string        `json:"description,omitempty"`
	DefaultState string        `json:"default_state,omitempty"`
	States       []ISMState    `json:"states"`
	ISMTemplate  []ISMTemplate `json:"ism_template,omitempty"`
}

// ISMState represents a state in an ISM policy
type ISMState struct {
	Name        string          `json:"name"`
	Actions     []ISMAction     `json:"actions,omitempty"`
	Transitions []ISMTransition `json:"transitions,omitempty"`
}

// ISMAction represents an action in an ISM state
type ISMAction struct {
	Retry         *ISMRetry               `json:"retry,omitempty"`
	Timeout       string                  `json:"timeout,omitempty"`
	Rollover      *ISMRolloverAction      `json:"rollover,omitempty"`
	Delete        *ISMDeleteAction        `json:"delete,omitempty"`
	ForceMerge    *ISMForceMergeAction    `json:"force_merge,omitempty"`
	ReadOnly      *ISMReadOnlyAction      `json:"read_only,omitempty"`
	ReplicaCount  *ISMReplicaCountAction  `json:"replica_count,omitempty"`
	IndexPriority *ISMIndexPriorityAction `json:"index_priority,omitempty"`
	Shrink        *ISMShrinkAction        `json:"shrink,omitempty"`
	Snapshot      *ISMSnapshotAction      `json:"snapshot,omitempty"`
	Close         *ISMCloseAction         `json:"close,omitempty"`
	Open          *ISMOpenAction          `json:"open,omitempty"`
	Notification  *ISMNotificationAction  `json:"notification,omitempty"`
	Allocation    *ISMAllocationAction    `json:"allocation,omitempty"`
	Custom        map[string]interface{}  `json:"custom,omitempty"`
}

// ISMRetry configures retry behavior for an action
type ISMRetry struct {
	Count   int    `json:"count"`
	Backoff string `json:"backoff,omitempty"` // "exponential", "constant", "linear"
	Delay   string `json:"delay,omitempty"`
}

// ISMRolloverAction configures rollover action
type ISMRolloverAction struct {
	MinSize             string `json:"min_size,omitempty"`
	MinDocs             int64  `json:"min_doc_count,omitempty"`
	MinAge              string `json:"min_index_age,omitempty"`
	MinPrimaryShardSize string `json:"min_primary_shard_size,omitempty"`
}

// ISMDeleteAction configures delete action
type ISMDeleteAction struct{}

// ISMForceMergeAction configures force merge action
type ISMForceMergeAction struct {
	MaxNumSegments int `json:"max_num_segments"`
}

// ISMReadOnlyAction configures read only action
type ISMReadOnlyAction struct{}

// ISMReplicaCountAction configures replica count action
type ISMReplicaCountAction struct {
	NumberOfReplicas int `json:"number_of_replicas"`
}

// ISMIndexPriorityAction configures index priority action
type ISMIndexPriorityAction struct {
	Priority int `json:"priority"`
}

// ISMShrinkAction configures shrink action
type ISMShrinkAction struct {
	NumNewShards             int     `json:"num_new_shards,omitempty"`
	PercentageOfSourceShards float64 `json:"percentage_of_source_shards,omitempty"`
	TargetIndexNameTemplate  string  `json:"target_index_name_template,omitempty"`
}

// ISMSnapshotAction configures snapshot action
type ISMSnapshotAction struct {
	Repository string `json:"repository"`
	Snapshot   string `json:"snapshot"`
}

// ISMCloseAction configures close action
type ISMCloseAction struct{}

// ISMOpenAction configures open action
type ISMOpenAction struct{}

// ISMNotificationAction configures notification action
type ISMNotificationAction struct {
	Destination     string                 `json:"destination"`
	MessageTemplate map[string]interface{} `json:"message_template,omitempty"`
}

// ISMAllocationAction configures allocation action
type ISMAllocationAction struct {
	Require map[string]string `json:"require,omitempty"`
	Include map[string]string `json:"include,omitempty"`
	Exclude map[string]string `json:"exclude,omitempty"`
	WaitFor string            `json:"wait_for,omitempty"`
}

// ISMTransition represents a transition to another state
type ISMTransition struct {
	StateName  string         `json:"state_name"`
	Conditions *ISMConditions `json:"conditions,omitempty"`
}

// ISMConditions represents conditions for a transition
type ISMConditions struct {
	MinAge  string            `json:"min_index_age,omitempty"`
	MinDocs int64             `json:"min_doc_count,omitempty"`
	MinSize string            `json:"min_size,omitempty"`
	Cron    *ISMCronCondition `json:"cron,omitempty"`
}

// ISMCronCondition represents a cron-based condition
type ISMCronCondition struct {
	Expression string `json:"expression"`
	Timezone   string `json:"timezone"`
}

// ISMTemplate represents a template for automatically applying ISM policies
type ISMTemplate struct {
	IndexPatterns []string `json:"index_patterns"`
	Priority      int      `json:"priority,omitempty"`
}

// ISMExplainResult represents the result of an ISM explain request
type ISMExplainResult struct {
	Index             string                 `json:"index"`
	IndexUUID         string                 `json:"index_uuid"`
	PolicyID          string                 `json:"policy_id"`
	PolicySeqNo       int64                  `json:"policy_seq_no"`
	PolicyPrimaryTerm int64                  `json:"policy_primary_term"`
	RolledOver        bool                   `json:"rolled_over"`
	State             map[string]interface{} `json:"state"`
	Info              map[string]interface{} `json:"info"`
}

// ISMPolicyBuilder provides a fluent API for creating ISM policies
type ISMPolicyBuilder struct {
	ctx          context.Context
	name         string
	description  string
	defaultState string
	states       []ISMState
	ismTemplate  []ISMTemplate
	errors       []error
}

// NewISMPolicyBuilder creates a new ISM policy builder
func NewISMPolicyBuilder(ctx context.Context, name string) *ISMPolicyBuilder {
	return &ISMPolicyBuilder{
		ctx:    ctx,
		name:   name,
		states: []ISMState{},
		errors: []error{},
	}
}

// Description sets the policy description
func (b *ISMPolicyBuilder) Description(desc string) *ISMPolicyBuilder {
	b.description = desc
	return b
}

// DefaultState sets the default state
func (b *ISMPolicyBuilder) DefaultState(state string) *ISMPolicyBuilder {
	b.defaultState = state
	return b
}

// ISMTemplate adds an ISM template for auto-attaching the policy
func (b *ISMPolicyBuilder) ISMTemplate(patterns []string, priority int) *ISMPolicyBuilder {
	b.ismTemplate = append(b.ismTemplate, ISMTemplate{
		IndexPatterns: patterns,
		Priority:      priority,
	})
	return b
}

// AddState adds a state to the policy and returns a state builder
func (b *ISMPolicyBuilder) AddState(name string) *ISMStateBuilder {
	return &ISMStateBuilder{
		parent: b,
		state: ISMState{
			Name:        name,
			Actions:     []ISMAction{},
			Transitions: []ISMTransition{},
		},
	}
}

// Build returns the ISM policy
func (b *ISMPolicyBuilder) Build() (ISMPolicy, error) {
	if len(b.errors) > 0 {
		return ISMPolicy{}, fmt.Errorf("policy builder has %d errors: %v", len(b.errors), b.errors)
	}

	if b.name == "" {
		return ISMPolicy{}, fmt.Errorf("policy name is required")
	}

	if len(b.states) == 0 {
		return ISMPolicy{}, fmt.Errorf("at least one state is required")
	}

	if b.defaultState == "" {
		// Use the first state as default
		b.defaultState = b.states[0].Name
	}

	return ISMPolicy{
		PolicyID:     b.name,
		Description:  b.description,
		DefaultState: b.defaultState,
		States:       b.states,
		ISMTemplate:  b.ismTemplate,
	}, nil
}

// BuildWithErrors returns the policy and any accumulated errors
func (b *ISMPolicyBuilder) BuildWithErrors() (ISMPolicy, []error) {
	if b.name == "" {
		b.errors = append(b.errors, fmt.Errorf("policy name is required"))
	}
	if len(b.states) == 0 {
		b.errors = append(b.errors, fmt.Errorf("at least one state is required"))
	}

	if b.defaultState == "" && len(b.states) > 0 {
		b.defaultState = b.states[0].Name
	}

	return ISMPolicy{
		PolicyID:     b.name,
		Description:  b.description,
		DefaultState: b.defaultState,
		States:       b.states,
		ISMTemplate:  b.ismTemplate,
	}, b.errors
}

// Ensure creates or updates the ISM policy (idempotent)
func (b *ISMPolicyBuilder) Ensure() error {
	policy, err := b.Build()
	if err != nil {
		return err
	}

	// Check if policy exists
	exists, _, err := getISMPolicyWithSeqNo(b.ctx, b.name)
	if err != nil {
		return fmt.Errorf("failed to check policy existence: %w", err)
	}

	body := map[string]interface{}{
		"policy": map[string]interface{}{
			"description":   policy.Description,
			"default_state": policy.DefaultState,
			"states":        policy.States,
		},
	}

	if len(policy.ISMTemplate) > 0 {
		body["policy"].(map[string]interface{})["ism_template"] = policy.ISMTemplate
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	var path string
	var method string
	if exists {
		// Update existing policy
		path = fmt.Sprintf("/_plugins/_ism/policies/%s", b.name)
		method = "PUT"
	} else {
		// Create new policy
		path = fmt.Sprintf("/_plugins/_ism/policies/%s", b.name)
		method = "PUT"
	}

	req, err := http.NewRequestWithContext(b.ctx, method, path, bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to create/update policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create/update policy: %s", string(bodyBytes))
	}

	return nil
}

// ISMStateBuilder provides a fluent API for building ISM states
type ISMStateBuilder struct {
	parent *ISMPolicyBuilder
	state  ISMState
}

// RolloverAction adds a rollover action to the state
func (s *ISMStateBuilder) RolloverAction(conditions RolloverConditions) *ISMStateBuilder {
	action := ISMAction{
		Rollover: &ISMRolloverAction{},
	}
	if conditions.MinAge != "" {
		action.Rollover.MinAge = conditions.MinAge
	}
	if conditions.MinDocs > 0 {
		action.Rollover.MinDocs = conditions.MinDocs
	}
	if conditions.MinSize != "" {
		action.Rollover.MinSize = conditions.MinSize
	}
	if conditions.MinPrimaryShardSize != "" {
		action.Rollover.MinPrimaryShardSize = conditions.MinPrimaryShardSize
	}
	s.state.Actions = append(s.state.Actions, action)
	return s
}

// DeleteAction adds a delete action to the state
func (s *ISMStateBuilder) DeleteAction() *ISMStateBuilder {
	s.state.Actions = append(s.state.Actions, ISMAction{
		Delete: &ISMDeleteAction{},
	})
	return s
}

// ForceMergeAction adds a force merge action to the state
func (s *ISMStateBuilder) ForceMergeAction(maxSegments int) *ISMStateBuilder {
	s.state.Actions = append(s.state.Actions, ISMAction{
		ForceMerge: &ISMForceMergeAction{
			MaxNumSegments: maxSegments,
		},
	})
	return s
}

// ReadOnlyAction adds a read only action to the state
func (s *ISMStateBuilder) ReadOnlyAction() *ISMStateBuilder {
	s.state.Actions = append(s.state.Actions, ISMAction{
		ReadOnly: &ISMReadOnlyAction{},
	})
	return s
}

// ReplicaCountAction adds a replica count action to the state
func (s *ISMStateBuilder) ReplicaCountAction(replicas int) *ISMStateBuilder {
	s.state.Actions = append(s.state.Actions, ISMAction{
		ReplicaCount: &ISMReplicaCountAction{
			NumberOfReplicas: replicas,
		},
	})
	return s
}

// IndexPriorityAction adds an index priority action to the state
func (s *ISMStateBuilder) IndexPriorityAction(priority int) *ISMStateBuilder {
	s.state.Actions = append(s.state.Actions, ISMAction{
		IndexPriority: &ISMIndexPriorityAction{
			Priority: priority,
		},
	})
	return s
}

// CloseAction adds a close action to the state
func (s *ISMStateBuilder) CloseAction() *ISMStateBuilder {
	s.state.Actions = append(s.state.Actions, ISMAction{
		Close: &ISMCloseAction{},
	})
	return s
}

// OpenAction adds an open action to the state
func (s *ISMStateBuilder) OpenAction() *ISMStateBuilder {
	s.state.Actions = append(s.state.Actions, ISMAction{
		Open: &ISMOpenAction{},
	})
	return s
}

// SnapshotAction adds a snapshot action to the state
func (s *ISMStateBuilder) SnapshotAction(repository, snapshot string) *ISMStateBuilder {
	s.state.Actions = append(s.state.Actions, ISMAction{
		Snapshot: &ISMSnapshotAction{
			Repository: repository,
			Snapshot:   snapshot,
		},
	})
	return s
}

// ShrinkAction adds a shrink action to the state
func (s *ISMStateBuilder) ShrinkAction(numShards int) *ISMStateBuilder {
	s.state.Actions = append(s.state.Actions, ISMAction{
		Shrink: &ISMShrinkAction{
			NumNewShards: numShards,
		},
	})
	return s
}

// AllocationAction adds an allocation action to the state
func (s *ISMStateBuilder) AllocationAction(require, include, exclude map[string]string) *ISMStateBuilder {
	s.state.Actions = append(s.state.Actions, ISMAction{
		Allocation: &ISMAllocationAction{
			Require: require,
			Include: include,
			Exclude: exclude,
		},
	})
	return s
}

// CustomAction adds a custom action to the state
func (s *ISMStateBuilder) CustomAction(action map[string]interface{}) *ISMStateBuilder {
	s.state.Actions = append(s.state.Actions, ISMAction{
		Custom: action,
	})
	return s
}

// TransitionTo adds a transition to another state
func (s *ISMStateBuilder) TransitionTo(stateName string) *ISMTransitionBuilder {
	return &ISMTransitionBuilder{
		parent: s,
		transition: ISMTransition{
			StateName: stateName,
		},
	}
}

// Done finalizes the state and returns to the parent builder
func (s *ISMStateBuilder) Done() *ISMPolicyBuilder {
	s.parent.states = append(s.parent.states, s.state)
	return s.parent
}

// ISMTransitionBuilder provides a fluent API for building ISM transitions
type ISMTransitionBuilder struct {
	parent     *ISMStateBuilder
	transition ISMTransition
}

// AfterMinAge sets the minimum age condition for the transition
func (t *ISMTransitionBuilder) AfterMinAge(age string) *ISMTransitionBuilder {
	if t.transition.Conditions == nil {
		t.transition.Conditions = &ISMConditions{}
	}
	t.transition.Conditions.MinAge = age
	return t
}

// AfterMinDocs sets the minimum document count condition for the transition
func (t *ISMTransitionBuilder) AfterMinDocs(docs int64) *ISMTransitionBuilder {
	if t.transition.Conditions == nil {
		t.transition.Conditions = &ISMConditions{}
	}
	t.transition.Conditions.MinDocs = docs
	return t
}

// AfterMinSize sets the minimum size condition for the transition
func (t *ISMTransitionBuilder) AfterMinSize(size string) *ISMTransitionBuilder {
	if t.transition.Conditions == nil {
		t.transition.Conditions = &ISMConditions{}
	}
	t.transition.Conditions.MinSize = size
	return t
}

// OnCron sets a cron-based condition for the transition
func (t *ISMTransitionBuilder) OnCron(expr, timezone string) *ISMTransitionBuilder {
	if t.transition.Conditions == nil {
		t.transition.Conditions = &ISMConditions{}
	}
	t.transition.Conditions.Cron = &ISMCronCondition{
		Expression: expr,
		Timezone:   timezone,
	}
	return t
}

// Done finalizes the transition and returns to the state builder
func (t *ISMTransitionBuilder) Done() *ISMStateBuilder {
	t.parent.state.Transitions = append(t.parent.state.Transitions, t.transition)
	return t.parent
}

// Helper function to get ISM policy with sequence number
func getISMPolicyWithSeqNo(ctx context.Context, name string) (bool, int64, error) {
	path := fmt.Sprintf("/_plugins/_ism/policies/%s", name)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return false, 0, err
	}

	resp, err := client.Perform(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return false, 0, nil
	}

	if resp.StatusCode >= 400 {
		return false, 0, nil
	}

	var result struct {
		SeqNo int64 `json:"_seq_no"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return true, 0, nil
	}

	return true, result.SeqNo, nil
}

// GetISMPolicy retrieves an ISM policy by name
func GetISMPolicy(ctx context.Context, name string) (*ISMPolicy, error) {
	path := fmt.Sprintf("/_plugins/_ism/policies/%s", name)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get policy: %s", string(bodyBytes))
	}

	var result struct {
		ID     string    `json:"_id"`
		Policy ISMPolicy `json:"policy"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result.Policy.PolicyID = result.ID
	return &result.Policy, nil
}

// DeleteISMPolicy deletes an ISM policy
func DeleteISMPolicy(ctx context.Context, name string) error {
	path := fmt.Sprintf("/_plugins/_ism/policies/%s", name)
	req, err := http.NewRequestWithContext(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete policy: %s", string(bodyBytes))
	}

	return nil
}

// ListISMPolicies lists all ISM policies
func ListISMPolicies(ctx context.Context) ([]ISMPolicy, error) {
	path := "/_plugins/_ism/policies"
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list policies: %s", string(bodyBytes))
	}

	var result struct {
		Policies []struct {
			ID     string    `json:"_id"`
			Policy ISMPolicy `json:"policy"`
		} `json:"policies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	policies := make([]ISMPolicy, len(result.Policies))
	for i, p := range result.Policies {
		policies[i] = p.Policy
		policies[i].PolicyID = p.ID
	}

	return policies, nil
}

// ApplyISMPolicy applies an ISM policy to an index
func ApplyISMPolicy(ctx context.Context, index, policyName string) error {
	body := map[string]interface{}{
		"policy_id": policyName,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	path := fmt.Sprintf("/_plugins/_ism/add/%s", index)
	req, err := http.NewRequestWithContext(ctx, "POST", path, bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to apply policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to apply policy: %s", string(bodyBytes))
	}

	return nil
}

// RemoveISMPolicy removes an ISM policy from an index
func RemoveISMPolicy(ctx context.Context, index string) error {
	path := fmt.Sprintf("/_plugins/_ism/remove/%s", index)
	req, err := http.NewRequestWithContext(ctx, "POST", path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to remove policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove policy: %s", string(bodyBytes))
	}

	return nil
}

// ExplainISM returns the ISM status for an index
func ExplainISM(ctx context.Context, index string) (*ISMExplainResult, error) {
	path := fmt.Sprintf("/_plugins/_ism/explain/%s", index)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Perform(req)
	if err != nil {
		return nil, fmt.Errorf("failed to explain ISM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to explain ISM: %s", string(bodyBytes))
	}

	var result map[string]ISMExplainResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	for idx, explain := range result {
		explain.Index = idx
		return &explain, nil
	}

	return nil, nil
}

// RetryISM retries a failed ISM action for an index
func RetryISM(ctx context.Context, index string) error {
	body := map[string]interface{}{
		"state": "retry",
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	path := fmt.Sprintf("/_plugins/_ism/retry/%s", index)
	req, err := http.NewRequestWithContext(ctx, "POST", path, bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Perform(req)
	if err != nil {
		return fmt.Errorf("failed to retry ISM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to retry ISM: %s", string(bodyBytes))
	}

	return nil
}

// ISMPolicyExists checks if an ISM policy exists
func ISMPolicyExists(ctx context.Context, name string) (bool, error) {
	exists, _, err := getISMPolicyWithSeqNo(ctx, name)
	return exists, err
}
