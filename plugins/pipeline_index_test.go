package plugins

import "testing"

func pipe(name string, order int32, tenant string, dataTypes ...string) *Pipeline {
	return &Pipeline{Name: name, Order: order, TenantId: tenant, DataTypes: dataTypes}
}

func names(ps []*Pipeline) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.Name)
	}
	return out
}

func equal(a []string, b ...string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// A tenant that never reordered anything gets the order the files declare.
// This is the common case and must not require any configuration at all.
func TestATenantWithNoPreferenceGetsTheShippedOrder(t *testing.T) {
	all := []*Pipeline{pipe("azure", 20, ""), pipe("aws", 10, ""), pipe("bitdefender", 0, "")}

	got := names(orderedFor(all, "tenant-a", nil, nil))

	if !equal(got, "bitdefender", "aws", "azure") {
		t.Errorf("order = %v, want the file order", got)
	}
}

// The point of the whole change: two tenants asking for different orders of the
// same shipped pipelines, and each getting theirs.
func TestTwoTenantsGetTheirOwnOrder(t *testing.T) {
	all := []*Pipeline{pipe("aws", 10, ""), pipe("azure", 20, "")}

	a := names(orderedFor(all, "tenant-a", []string{"azure", "aws"}, nil))
	b := names(orderedFor(all, "tenant-b", nil, nil))

	if !equal(a, "azure", "aws") {
		t.Errorf("tenant A order = %v, want its own", a)
	}
	if !equal(b, "aws", "azure") {
		t.Errorf("tenant B order = %v, want the default", b)
	}
}

// A partial preference orders what it names and leaves the rest alone, so a
// tenant inserting one pipeline does not have to restate the other thirty-four.
func TestNamingSomePipelinesLeavesTheRestInFileOrder(t *testing.T) {
	all := []*Pipeline{pipe("aws", 10, ""), pipe("azure", 20, ""), pipe("gcp", 30, "")}

	got := names(orderedFor(all, "tenant-a", []string{"gcp"}, nil))

	if !equal(got, "gcp", "aws", "azure") {
		t.Errorf("order = %v, want the named one first then the file order", got)
	}
}

// A tenant's own pipeline runs for it and for nobody else — enrichment is the
// case this exists for.
func TestATenantsOwnPipelineIsPrivateToIt(t *testing.T) {
	all := []*Pipeline{pipe("aws", 10, ""), pipe("aws-enrich", 15, "tenant-a", "aws")}

	a := names(orderedFor(all, "tenant-a", nil, nil))
	b := names(orderedFor(all, "tenant-b", nil, nil))

	if !equal(a, "aws", "aws-enrich") {
		t.Errorf("tenant A = %v, want its enrichment after the shipped parser", a)
	}
	if !equal(b, "aws") {
		t.Errorf("tenant B = %v, want no sight of another tenant's pipeline", b)
	}
}

// Disabling is resolved here rather than per event, so it must actually remove
// the pipeline from the tenant's list.
func TestADisabledPipelineIsAbsent(t *testing.T) {
	all := []*Pipeline{pipe("aws", 10, ""), pipe("azure", 20, "")}

	got := names(orderedFor(all, "tenant-a", nil, []string{"azure"}))

	if !equal(got, "aws") {
		t.Errorf("order = %v, want azure gone", got)
	}
}

// A preference can outlive the pipeline it names. Skipping the stale entry is
// the only answer that still parses the tenant's logs.
func TestAPreferenceNamingAMissingPipelineIsIgnored(t *testing.T) {
	all := []*Pipeline{pipe("aws", 10, "")}

	got := names(orderedFor(all, "tenant-a", []string{"deleted-one", "aws"}, nil))

	if !equal(got, "aws") {
		t.Errorf("order = %v, want the surviving pipeline", got)
	}
}

// A disabled pipeline the tenant also ordered must stay disabled: the two
// settings are read from the same file and the stricter one has to win.
func TestOrderingADisabledPipelineDoesNotReviveIt(t *testing.T) {
	all := []*Pipeline{pipe("aws", 10, ""), pipe("azure", 20, "")}

	got := names(orderedFor(all, "tenant-a", []string{"azure", "aws"}, []string{"azure"}))

	if !equal(got, "aws") {
		t.Errorf("order = %v, want the disabled pipeline to stay out", got)
	}
}

// A tenant the configuration has not heard of yet still has to be parsed —
// answering with nothing would drop its logs silently.
func TestAnUnknownTenantFallsBackToTheDefaultOrder(t *testing.T) {
	buildPipelineIndex(
		[]*Pipeline{pipe("aws", 10, ""), pipe("azure", 20, "")},
		[]*Tenant{{Id: "tenant-a", PipelineOrder: []string{"azure", "aws"}}},
	)

	got := names(PipelinesFor("tenant-never-seen"))

	if !equal(got, "aws", "azure") {
		t.Errorf("unknown tenant = %v, want the default order", got)
	}
	if known := names(PipelinesFor("tenant-a")); !equal(known, "azure", "aws") {
		t.Errorf("known tenant = %v, want its own order", known)
	}
}
