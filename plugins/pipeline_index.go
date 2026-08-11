package plugins

import (
	"sort"
	"sync"
)

// DefaultPipelineOrder is the key holding the order a tenant gets when it has
// never reordered anything — and the answer for a tenant the configuration has
// not heard of yet. A draft whose tenant is missing must still be parsed;
// serving nothing would drop its logs without a word.
const DefaultPipelineOrder = ""

var (
	pipelineIndex   map[string][]*Pipeline
	pipelineIndexMu sync.RWMutex
)

// PipelinesFor returns the pipelines that apply to one tenant, already in the
// order that tenant wants them and with its disabled ones removed.
//
// Resolving this once per configuration load rather than per event is the point:
// the parsing loop used to walk every pipeline and re-check ownership and the
// disabled list for each draft.
func PipelinesFor(tenantId string) []*Pipeline {
	pipelineIndexMu.RLock()
	defer pipelineIndexMu.RUnlock()

	if list, ok := pipelineIndex[tenantId]; ok {
		return list
	}
	return pipelineIndex[DefaultPipelineOrder]
}

// buildPipelineIndex resolves the per-tenant view of the pipeline set. Callers
// must hold the configuration write lock.
func buildPipelineIndex(pipelines []*Pipeline, tenants []*Tenant) {
	index := make(map[string][]*Pipeline, len(tenants)+1)
	index[DefaultPipelineOrder] = orderedFor(pipelines, "", nil, nil)

	for _, t := range tenants {
		if t == nil || t.Id == "" {
			continue
		}
		index[t.Id] = orderedFor(pipelines, t.Id, t.PipelineOrder, t.DisabledPipelines)
	}

	pipelineIndexMu.Lock()
	pipelineIndex = index
	pipelineIndexMu.Unlock()
}

// orderedFor selects the pipelines one tenant may run and puts them in its
// order: the ones it named first, in exactly that sequence, then everything
// else by the order declared in its own file.
//
// A name the tenant listed that no longer exists is skipped rather than
// treated as an error — a pipeline can be deleted while a tenant's preference
// still mentions it, and refusing to parse anything would be a poor answer to
// a stale line in a config file.
func orderedFor(pipelines []*Pipeline, tenantId string, order, disabled []string) []*Pipeline {
	skip := make(map[string]bool, len(disabled))
	for _, name := range disabled {
		skip[name] = true
	}

	mine := make([]*Pipeline, 0, len(pipelines))
	byName := make(map[string]*Pipeline, len(pipelines))
	for _, p := range pipelines {
		if p == nil || skip[p.Name] {
			continue
		}
		// A pipeline owned by another tenant never applies here; one with no
		// owner ships with the release and applies to everybody.
		if p.TenantId != "" && p.TenantId != tenantId {
			continue
		}
		mine = append(mine, p)
		byName[p.Name] = p
	}

	sort.SliceStable(mine, func(i, j int) bool { return mine[i].Order < mine[j].Order })

	if len(order) == 0 {
		return mine
	}

	out := make([]*Pipeline, 0, len(mine))
	placed := make(map[string]bool, len(order))
	for _, name := range order {
		p, ok := byName[name]
		if !ok || placed[name] {
			continue
		}
		out = append(out, p)
		placed[name] = true
	}
	for _, p := range mine {
		if !placed[p.Name] {
			out = append(out, p)
		}
	}
	return out
}
