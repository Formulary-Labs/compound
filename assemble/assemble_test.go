package assemble_test

import (
	"strings"
	"testing"

	"github.com/Formulary-Labs/compound/assemble"
)

func TestAssemble_iso27001_structure(t *testing.T) {
	cfg := assemble.DocumentConfig{
		Program:  "test",
		Standard: assemble.ISO27001,
	}
	ctx := &assemble.ProgramContext{
		Program: "test",
		Scope:   "Product X cloud platform",
		Coverage: &struct {
			TotalControls int     `json:"total_controls,omitempty"`
			EvidencedPct  float64 `json:"evidenced_pct,omitempty"`
			GapPct        float64 `json:"gap_pct,omitempty"`
		}{TotalControls: 93, EvidencedPct: 75, GapPct: 10},
	}
	doc := assemble.Assemble(cfg, ctx)

	for _, clause := range []string{
		"Clause 4",
		"Clause 5",
		"Clause 6",
		"Clause 7",
		"Clause 8",
		"Clause 9",
		"Clause 10",
	} {
		if !strings.Contains(doc, clause) {
			t.Errorf("document missing %s", clause)
		}
	}

	if !strings.Contains(doc, "ISO/IEC 27001") {
		t.Error("document missing ISO/IEC 27001 reference")
	}
	if !strings.Contains(doc, "ISMS") {
		t.Error("document missing ISMS reference")
	}
	if !strings.Contains(doc, "Product X cloud platform") {
		t.Error("document missing scope statement")
	}
}

func TestAssemble_iso42001_aiClauses(t *testing.T) {
	cfg := assemble.DocumentConfig{
		Program:  "test",
		Standard: assemble.ISO42001,
	}
	ctx := &assemble.ProgramContext{Program: "test"}
	doc := assemble.Assemble(cfg, ctx)

	if !strings.Contains(doc, "AI Risk Assessment") {
		t.Error("ISO42001 document missing AI Risk Assessment clause")
	}
	if !strings.Contains(doc, "Statement of Applicability") {
		t.Error("ISO42001 document missing SoA reference")
	}
	if !strings.Contains(doc, "AIMS") {
		t.Error("ISO42001 document missing AIMS reference")
	}
}

func TestAssemble_iec62443(t *testing.T) {
	cfg := assemble.DocumentConfig{
		Program:  "test",
		Standard: assemble.IEC62443,
	}
	ctx := &assemble.ProgramContext{Program: "test"}
	doc := assemble.Assemble(cfg, ctx)

	if !strings.Contains(doc, "CSMS") {
		t.Error("IEC62443 document missing CSMS reference")
	}
}

func TestAssemble_dataNeedFlags(t *testing.T) {
	cfg := assemble.DocumentConfig{
		Program:  "test",
		Standard: assemble.ISO27001,
	}
	doc := assemble.Assemble(cfg, nil)

	if !strings.Contains(doc, "[DATA NEEDED") {
		t.Error("document missing [DATA NEEDED] flags for sections requiring narrative")
	}
}

func TestAssemble_brandedOutput(t *testing.T) {
	cfg := assemble.DocumentConfig{
		Program:       "test",
		Standard:      assemble.ISO27001,
		BrandedOutput: true,
		OrgName:       "Acme Corp",
	}
	doc := assemble.Assemble(cfg, &assemble.ProgramContext{Program: "test"})

	if !strings.Contains(doc, "Acme Corp") {
		t.Error("branded output missing organization name")
	}
}

func TestAssembleDelta_regeneratesOnlyChangedClauses(t *testing.T) {
	cfg := assemble.DocumentConfig{
		Program:    "test",
		Standard:   assemble.ISO27001,
		ReviewMode: assemble.DeltaReview,
	}

	prev := &assemble.ProgramContext{
		Program:       "test",
		Scope:         "Original scope",
		RiskCount:     10,
		OpenRisks:     5,
		CriticalRisks: 1,
	}
	curr := &assemble.ProgramContext{
		Program:       "test",
		Scope:         "Updated scope — new boundary",
		RiskCount:     15, // risk count changed
		OpenRisks:     8,
		CriticalRisks: 2,
	}

	doc := assemble.AssembleDelta(cfg, curr, prev)

	// Delta document must reference delta_review mode.
	if !strings.Contains(doc, "delta_review") {
		t.Error("delta output should identify itself as delta_review mode")
	}

	// Changed clauses (4: scope, 6: risks) must be present.
	if !strings.Contains(doc, "Clause 4") {
		t.Error("Clause 4 should be regenerated — scope changed")
	}
	if !strings.Contains(doc, "Clause 6") {
		t.Error("Clause 6 should be regenerated — risk count changed")
	}

	// Unchanged clauses (5, 7, 8, 9, 10) should NOT appear — no inputs changed.
	for _, unchanged := range []string{"Clause 5", "Clause 7", "Clause 9", "Clause 10"} {
		if strings.Contains(doc, unchanged) {
			t.Errorf("%s should not be regenerated — no inputs changed", unchanged)
		}
	}

	// Version control table must record updated clauses.
	if !strings.Contains(doc, "Version Control") {
		t.Error("delta output should include Version Control table")
	}
}

func TestAssembleDelta_noChangesReturnsNote(t *testing.T) {
	cfg := assemble.DocumentConfig{
		Program:    "test",
		Standard:   assemble.ISO27001,
		ReviewMode: assemble.DeltaReview,
	}

	identical := &assemble.ProgramContext{
		Program:   "test",
		Scope:     "Same scope",
		RiskCount: 10,
	}
	doc := assemble.AssembleDelta(cfg, identical, identical)

	if !strings.Contains(doc, "No clause inputs changed") {
		t.Error("when no inputs differ, delta should return a no-change note")
	}
}

func TestAssembleSection_regeneratesSingleClause(t *testing.T) {
	cfg := assemble.DocumentConfig{
		Program:       "test",
		Standard:      assemble.ISO27001,
		ReviewMode:    assemble.SectionUpdate,
		SectionTarget: "8",
	}
	ctx := &assemble.ProgramContext{
		Program: "test",
		Coverage: &struct {
			TotalControls int     `json:"total_controls,omitempty"`
			EvidencedPct  float64 `json:"evidenced_pct,omitempty"`
			GapPct        float64 `json:"gap_pct,omitempty"`
		}{TotalControls: 93, EvidencedPct: 82, GapPct: 8},
	}

	doc, err := assemble.AssembleSection(cfg, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(doc, "Clause 8") {
		t.Error("section_update for section 8 should contain Clause 8")
	}
	if !strings.Contains(doc, "section_update") {
		t.Error("section output should identify itself as section_update mode")
	}
	// Only Clause 8 — other clauses must not appear.
	for _, other := range []string{"Clause 4", "Clause 5", "Clause 6", "Clause 7", "Clause 9", "Clause 10"} {
		if strings.Contains(doc, other) {
			t.Errorf("%s should not appear in section_update for clause 8", other)
		}
	}
}

func TestAssembleSection_invalidSectionReturnsError(t *testing.T) {
	cfg := assemble.DocumentConfig{
		Program:       "test",
		Standard:      assemble.ISO27001,
		ReviewMode:    assemble.SectionUpdate,
		SectionTarget: "99",
	}
	_, err := assemble.AssembleSection(cfg, nil)
	if err == nil {
		t.Error("expected error for invalid section number 99")
	}
}
