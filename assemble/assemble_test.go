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
