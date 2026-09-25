package assemble_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Formulary-Labs/compound/assemble"
	"github.com/Formulary-Labs/substrate/artifact"
)

func testPolicy() *artifact.Policy {
	aff := "Security"
	p := &artifact.Policy{
		Title: "Information Security Policy",
	}
	p.Metadata.Id = "policy-clause-5.2"
	p.Metadata.Description = "Protect organizational information assets through documented controls."
	p.Metadata.Author.Name = "Security Team"
	p.Metadata.ApplicabilityGroups = []artifact.Group{
		{Id: "clause-5.2", Title: "Policy"},
	}
	p.Scope.In.Technologies = []string{"Cloud Computing", "Web Applications"}
	p.Scope.In.Geopolitical = []string{"United States", "European Union"}
	p.Scope.Out.Technologies = []string{"Legacy Systems"}
	p.Contacts.Responsible = []artifact.Contact{{Name: "IT Director", Affiliation: &aff}}
	p.Contacts.Accountable = []artifact.Contact{{Name: "CISO", Affiliation: &aff}}
	p.Adherence.NonCompliance = "Non-compliance is escalated to the CISO within 5 business days."
	return p
}

func testCatalog() *artifact.ControlCatalog {
	cat := &artifact.ControlCatalog{
		Title: "ISO/IEC 27001:2022 Control Catalog",
		Groups: []artifact.Group{
			{Id: "A.5", Title: "Organizational Controls"},
			{Id: "A.8", Title: "Technological Controls"},
		},
		Controls: []artifact.Control{
			{Id: "A.5.1", Title: "Policies for information security"},
		},
	}
	cat.Metadata.Id = "iso27001-catalog"
	cat.Metadata.Description = "ISO/IEC 27001:2022"
	return cat
}

func TestAssemblePlan_goldenBody(t *testing.T) {
	cfg := assemble.DocumentConfig{
		Program:     "test-isms",
		Standard:    assemble.ISO27001,
		GeneratedAt: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
	}
	set := &assemble.GemaraSet{
		Policies: []assemble.PolicyRef{
			{Path: "testdata/policy.yaml", Policy: testPolicy(), Clause: "5.2"},
		},
		Catalog:     testCatalog(),
		CatalogPath: "testdata/catalog.yaml",
		ArtifactHashes: map[string]string{
			"testdata/policy.yaml":  "aaa",
			"testdata/catalog.yaml": "bbb",
		},
	}

	result := assemble.AssemblePlan(cfg, set)
	doc := result.Markdown

	mustContain := []string{
		"ISMS",
		"Security Plan",
		"ISO/IEC 27001:2022",
		"**Generated:** 2026-09-25",
		"Clause 4",
		"Clause 5",
		"**PURPOSE**",
		"Protect organizational information assets",
		"**SCOPE**",
		"Cloud Computing",
		"**POLICY DETAILS**",
		"Non-compliance is escalated",
		"IT Director",
		"CISO",
		"Organizational Controls",
		"assessment packet",
		"Assembly Provenance",
	}
	for _, s := range mustContain {
		if !strings.Contains(doc, s) {
			t.Errorf("golden body missing %q", s)
		}
	}

	// Hard packet boundary: no risk count grids.
	if strings.Contains(doc, "risk entries") {
		t.Error("plan body must not inline risk count grids")
	}

	// Determinism: same inputs → same body.
	again := assemble.AssemblePlan(cfg, set)
	if again.Markdown != doc {
		t.Error("AssemblePlan is not deterministic for fixed GeneratedAt")
	}

	if !result.Report.Clauses[1].Filled { // clause 5
		t.Error("expected clause 5 to be marked filled")
	}
}

func TestAssemblePlan_packetBoundaryRiskCitation(t *testing.T) {
	cfg := assemble.DocumentConfig{
		Program:     "test",
		Standard:    assemble.ISO27001,
		GeneratedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	rc := &artifact.RiskCatalog{Title: "Org Risks"}
	rc.Metadata.Id = "risks-001"
	set := &assemble.GemaraSet{
		Policies: []assemble.PolicyRef{
			{Path: "p.yaml", Policy: testPolicy(), Clause: "5.2"},
		},
		RiskCatalogs: []assemble.RiskCatalogRef{
			{Path: "risks.yaml", Catalog: rc},
		},
	}
	result := assemble.AssemblePlan(cfg, set)
	if !strings.Contains(result.Markdown, "RiskCatalog `risks-001`") {
		t.Error("expected RiskCatalog citation in clause 6")
	}
	if !strings.Contains(result.Markdown, "not reproduced in this plan") {
		t.Error("expected packet boundary language")
	}
	if len(result.Report.CitationsOnly) != 1 {
		t.Errorf("expected 1 citations_only entry, got %d", len(result.Report.CitationsOnly))
	}
}

func TestAssemblePlanDelta_noChange(t *testing.T) {
	cfg := assemble.DocumentConfig{Program: "test", Standard: assemble.ISO27001}
	set := &assemble.GemaraSet{
		ArtifactHashes: map[string]string{"a.yaml": "deadbeef"},
	}
	result := assemble.AssemblePlanDelta(cfg, set, set)
	if !strings.Contains(result.Markdown, "No gemara inputs changed") {
		t.Error("expected no-change note when fingerprints match")
	}
}

func TestResolveClauseViaApplicabilityGroup(t *testing.T) {
	p := testPolicy()
	set := &assemble.GemaraSet{
		Policies: []assemble.PolicyRef{
			{Path: "p.yaml", Policy: p, Clause: "5.2"},
		},
	}
	got := set.PoliciesForClause("5.2")
	if len(got) != 1 {
		t.Fatalf("expected 1 policy for 5.2, got %d", len(got))
	}
}
