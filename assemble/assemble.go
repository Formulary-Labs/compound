// Package assemble implements Annex SL-structured management system document
// generation for compound.
//
// compound is the deterministic scaffolding layer: it assembles the structure
// of an ISMS/AIMS/CSMS document (Clauses 4-10) from program artifacts, fills
// sections with data where the content is deterministic, and flags sections
// that require LLM narration or manual review.
//
// Narration (prose synthesis, tone calibration, stakeholder-specific language)
// is explicitly NOT in scope for this tool — those sections are flagged with
// [DATA NEEDED: narrative — agent layer] to be completed by the orchestrator.
//
// Standard support: iso27001 (ISMS), iso42001 (AIMS), iec62443 (CSMS).
package assemble

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Standard is the management system standard.
type Standard string

//nolint:revive // Standard constants are self-documenting string identifiers.
const (
	ISO27001 Standard = "iso27001"
	ISO42001 Standard = "iso42001"
	IEC62443 Standard = "iec62443"
)

// ReviewMode controls how the document is assembled.
type ReviewMode string

//nolint:revive // ReviewMode constants are self-documenting string identifiers.
const (
	FullAssembly  ReviewMode = "full_assembly"
	DeltaReview   ReviewMode = "delta_review"
	SectionUpdate ReviewMode = "section_update"
)

// DocumentConfig controls compound's assembly behavior.
type DocumentConfig struct {
	Program       string     `json:"program"`
	Standard      Standard   `json:"standard"`
	OutputName    string     `json:"output_name,omitempty"`
	Applicability string     `json:"applicability,omitempty"`
	ReviewCadence string     `json:"review_cadence,omitempty"` // annual | biannual | quarterly
	ReviewMode    ReviewMode `json:"review_mode,omitempty"`
	BrandedOutput bool       `json:"branded_output,omitempty"`
	OrgName       string     `json:"org_name,omitempty"`
	SectionTarget string     `json:"section_target,omitempty"` // for section_update mode
}

// ProgramContext is the data compound reads from program artifacts.
type ProgramContext struct {
	Program  string `json:"program"`
	Standard string `json:"standard,omitempty"`

	// Scope and context.
	Scope       string `json:"scope,omitempty"`
	ProductName string `json:"product_name,omitempty"`
	Framework   string `json:"framework,omitempty"`

	// Coverage data.
	Coverage *struct {
		TotalControls int     `json:"total_controls,omitempty"`
		EvidencedPct  float64 `json:"evidenced_pct,omitempty"`
		GapPct        float64 `json:"gap_pct,omitempty"`
	} `json:"coverage,omitempty"`

	// Risk data.
	RiskCount     int `json:"risk_count,omitempty"`
	OpenRisks     int `json:"open_risks,omitempty"`
	CriticalRisks int `json:"critical_risks,omitempty"`

	// Run metadata.
	LastUpdated *time.Time `json:"last_updated,omitempty"`
	RunDate     *time.Time `json:"run_date,omitempty"`

	// Stakeholders.
	Owner string `json:"owner,omitempty"`
}

// LoadProgramContext reads program context from a run state JSON.
func LoadProgramContext(path string) (*ProgramContext, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading program context %q: %w", path, err)
	}
	var ctx ProgramContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, fmt.Errorf("parsing program context %q: %w", path, err)
	}
	return &ctx, nil
}

// Assemble produces the Annex SL-structured management system document.
// For delta_review mode, use AssembleDelta. For section_update mode, use AssembleSection.
func Assemble(cfg DocumentConfig, ctx *ProgramContext) string {
	if ctx == nil {
		ctx = &ProgramContext{Program: cfg.Program}
	}
	if string(cfg.Standard) == "" {
		cfg.Standard = Standard(ctx.Standard)
	}

	docName := cfg.OutputName
	if docName == "" {
		docName = standardDocName(cfg.Standard)
	}
	orgName := "the Organization"
	if cfg.BrandedOutput && cfg.OrgName != "" {
		orgName = cfg.OrgName
	}

	sb := &strings.Builder{}
	writeHeader(sb, docName, cfg, ctx, orgName)
	writeClause4(sb, cfg, ctx, orgName)
	writeClause5(sb, cfg, ctx, orgName)
	writeClause6(sb, cfg, ctx)
	writeClause7(sb, cfg, ctx, orgName)
	writeClause8(sb, cfg, ctx)
	writeClause9(sb, cfg, ctx, orgName)
	writeClause10(sb, cfg, ctx)
	writeVersionControl(sb, cfg, ctx)
	return sb.String()
}

func writeHeader(sb *strings.Builder, docName string, cfg DocumentConfig, ctx *ProgramContext, orgName string) {
	fmt.Fprintf(sb, "# %s\n\n", docName)
	fmt.Fprintf(sb, "**Standard:** %s  \n", standardDisplay(cfg.Standard))
	fmt.Fprintf(sb, "**Program:** %s  \n", ctx.Program)
	fmt.Fprintf(sb, "**Organization:** %s  \n", orgName)
	if cfg.Applicability != "" {
		fmt.Fprintf(sb, "**Scope:** %s  \n", cfg.Applicability)
	} else if ctx.Scope != "" {
		fmt.Fprintf(sb, "**Scope:** %s  \n", ctx.Scope)
	}
	cadence := cfg.ReviewCadence
	if cadence == "" {
		cadence = "annual"
	}
	fmt.Fprintf(sb, "**Review Cadence:** %s  \n", cadence)
	fmt.Fprintf(sb, "**Generated:** %s\n\n", time.Now().Format("2006-01-02"))
	fmt.Fprintf(sb, "---\n\n")
	fmt.Fprintf(sb, "> This document was assembled by [compound](https://github.com/Formulary-Labs/compound). Sections marked `[DATA NEEDED: narrative]` require review and completion by the responsible compliance manager before this document is considered final.\n\n")
	fmt.Fprintf(sb, "---\n\n")
}

func writeClause4(sb *strings.Builder, cfg DocumentConfig, ctx *ProgramContext, orgName string) {
	fmt.Fprintf(sb, "## Clause 4 — Context of the Organization\n\n")
	fmt.Fprintf(sb, "### 4.1 Understanding the Organization and Its Context\n\n")
	if ctx.Scope != "" || ctx.ProductName != "" {
		fmt.Fprintf(sb, "%s operates %s within the following context:\n\n", orgName, ctx.ProductName)
	}
	fmt.Fprintf(sb, "[DATA NEEDED: narrative — describe the internal and external factors relevant to the %s, including relevant strategic, operational, regulatory, and technological context.]\n\n", standardDocName(cfg.Standard))
	fmt.Fprintf(sb, "### 4.2 Understanding the Needs and Expectations of Interested Parties\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: narrative — list relevant interested parties (customers, regulators, auditors, employees, vendors) and their relevant requirements.]\n\n")
	fmt.Fprintf(sb, "### 4.3 Determining the Scope of the %s\n\n", shortStandardName(cfg.Standard))
	if cfg.Applicability != "" {
		fmt.Fprintf(sb, "**Scope statement:** %s\n\n", cfg.Applicability)
	} else if ctx.Scope != "" {
		fmt.Fprintf(sb, "**Scope statement:** %s\n\n", ctx.Scope)
	} else {
		fmt.Fprintf(sb, "[DATA NEEDED: scope — provide a clear boundary statement for the %s, including what is included, what is excluded, and why.]\n\n", shortStandardName(cfg.Standard))
	}
	fmt.Fprintf(sb, "### 4.4 %s\n\n", shortStandardName(cfg.Standard))
	fmt.Fprintf(sb, "%s has established, implemented, maintains, and continually improves a %s in accordance with the requirements of %s.\n\n",
		orgName, standardDocName(cfg.Standard), standardDisplay(cfg.Standard))
}

func writeClause5(sb *strings.Builder, cfg DocumentConfig, ctx *ProgramContext, _ string) {
	fmt.Fprintf(sb, "## Clause 5 — Leadership\n\n")
	fmt.Fprintf(sb, "### 5.1 Leadership and Commitment\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: narrative — describe how top management demonstrates leadership and commitment to the %s.]\n\n", shortStandardName(cfg.Standard))
	fmt.Fprintf(sb, "### 5.2 Policy\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: policy text — provide the %s policy, including commitments to satisfy applicable requirements, continual improvement, and achievement of %s objectives.]\n\n",
		shortStandardName(cfg.Standard), shortStandardName(cfg.Standard))
	fmt.Fprintf(sb, "### 5.3 Organizational Roles, Responsibilities, and Authorities\n\n")
	if ctx.Owner != "" {
		fmt.Fprintf(sb, "**Program Owner:** %s\n\n", ctx.Owner)
	}
	fmt.Fprintf(sb, "[DATA NEEDED: RACI — list the roles responsible for %s management, including the person responsible for reporting %s performance to top management.]\n\n",
		shortStandardName(cfg.Standard), shortStandardName(cfg.Standard))
}

func writeClause6(sb *strings.Builder, cfg DocumentConfig, ctx *ProgramContext) {
	fmt.Fprintf(sb, "## Clause 6 — Planning\n\n")
	fmt.Fprintf(sb, "### 6.1 Actions to Address Risks and Opportunities\n\n")
	if ctx.RiskCount > 0 {
		fmt.Fprintf(sb, "The risk register currently contains **%d risk entries** (%d open", ctx.RiskCount, ctx.OpenRisks)
		if ctx.CriticalRisks > 0 {
			fmt.Fprintf(sb, ", **%d critical**", ctx.CriticalRisks)
		}
		fmt.Fprintf(sb, ").\n\n")
	}
	fmt.Fprintf(sb, "[DATA NEEDED: narrative — describe the process for identifying, assessing, and treating risks and opportunities relevant to the %s. Reference the risk register maintained via specimen.]\n\n", shortStandardName(cfg.Standard))
	fmt.Fprintf(sb, "### 6.2 %s Objectives and Planning to Achieve Them\n\n", shortStandardName(cfg.Standard))
	fmt.Fprintf(sb, "[DATA NEEDED: objectives — list the %s objectives for the current cycle, with owners, measures, and target dates.]\n\n", shortStandardName(cfg.Standard))
	if cfg.Standard == ISO27001 || cfg.Standard == ISO42001 {
		fmt.Fprintf(sb, "### 6.3 Planning of Changes\n\n")
		fmt.Fprintf(sb, "[DATA NEEDED: narrative — describe how planned changes to the %s are managed.]\n\n", shortStandardName(cfg.Standard))
	}
}

func writeClause7(sb *strings.Builder, cfg DocumentConfig, ctx *ProgramContext, orgName string) {
	fmt.Fprintf(sb, "## Clause 7 — Support\n\n")
	fmt.Fprintf(sb, "### 7.1 Resources\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: narrative — describe the resources provided to support the %s, including personnel, tooling, and budget allocation.]\n\n", shortStandardName(cfg.Standard))
	fmt.Fprintf(sb, "### 7.2 Competence\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: narrative — describe how %s ensures that persons affecting %s performance are competent, and how competence is maintained.]\n\n", orgName, shortStandardName(cfg.Standard))
	fmt.Fprintf(sb, "### 7.3 Awareness\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: narrative — describe the awareness program, including how personnel are made aware of the %s policy and their contribution to its effectiveness.]\n\n", shortStandardName(cfg.Standard))
	fmt.Fprintf(sb, "### 7.4 Communication\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: communication plan — describe what is communicated, when, to whom, and through what channels.]\n\n")
	fmt.Fprintf(sb, "### 7.5 Documented Information\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: document inventory — list the documented information required by %s and maintained by %s, with version, owner, and retention period.]\n\n",
		standardDisplay(cfg.Standard), orgName)
}

func writeClause8(sb *strings.Builder, cfg DocumentConfig, ctx *ProgramContext) {
	fmt.Fprintf(sb, "## Clause 8 — Operation\n\n")
	fmt.Fprintf(sb, "### 8.1 Operational Planning and Control\n\n")
	if ctx.Coverage != nil {
		fmt.Fprintf(sb, "Current control coverage: **%.0f%% evidenced** / **%.0f%% gap** across **%d controls**.\n\n",
			ctx.Coverage.EvidencedPct, ctx.Coverage.GapPct, ctx.Coverage.TotalControls)
	}
	fmt.Fprintf(sb, "[DATA NEEDED: narrative — describe the operational processes for planning and controlling the processes needed to meet %s requirements.]\n\n", shortStandardName(cfg.Standard))

	if cfg.Standard == ISO42001 {
		fmt.Fprintf(sb, "### 8.2 AI Risk Assessment\n\n")
		fmt.Fprintf(sb, "[DATA NEEDED: AI risk assessment summary — describe the AI-specific risk assessment process, including identification of AI-specific risks (bias, explainability, data quality, model drift).]\n\n")
		fmt.Fprintf(sb, "### 8.3 AI Risk Treatment\n\n")
		fmt.Fprintf(sb, "[DATA NEEDED: AI risk treatment plan — describe the treatment options applied to identified AI risks.]\n\n")
		fmt.Fprintf(sb, "### 8.4 Statement of Applicability\n\n")
		fmt.Fprintf(sb, "The Statement of Applicability is maintained as a structured artifact and generated deterministically via [formula](https://github.com/Formulary-Labs/formula). Current version available at `data/%s/soa.csv`.\n\n", ctx.Program)
	} else if cfg.Standard == ISO27001 {
		fmt.Fprintf(sb, "### 8.2 Information Security Risk Assessment\n\n")
		fmt.Fprintf(sb, "[DATA NEEDED: risk assessment — reference the current risk assessment output from specimen and formula.]\n\n")
		fmt.Fprintf(sb, "### 8.3 Information Security Risk Treatment\n\n")
		fmt.Fprintf(sb, "[DATA NEEDED: risk treatment plan — describe the treatment options applied to identified information security risks.]\n\n")
	}
}

func writeClause9(sb *strings.Builder, _ DocumentConfig, _ *ProgramContext, _ string) {
	fmt.Fprintf(sb, "## Clause 9 — Performance Evaluation\n\n")
	fmt.Fprintf(sb, "### 9.1 Monitoring, Measurement, Analysis, and Evaluation\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: KPI/KRI table — list what is monitored, how, by whom, when, and how results are analyzed. Reference titer for coverage metrics and specimen for risk posture.]\n\n")
	fmt.Fprintf(sb, "### 9.2 Internal Audit\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: audit program — describe the internal audit schedule, scope, criteria, frequency, and how audit results are reported to management.]\n\n")
	fmt.Fprintf(sb, "### 9.3 Management Review\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: management review records — describe the management review process, including frequency, inputs (from vital/decay), and outputs. Last management review: [DATE].]\n\n")
}

func writeClause10(sb *strings.Builder, cfg DocumentConfig, _ *ProgramContext) {
	fmt.Fprintf(sb, "## Clause 10 — Improvement\n\n")
	fmt.Fprintf(sb, "### 10.1 Continual Improvement\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: narrative — describe the process for continually improving the suitability, adequacy, and effectiveness of the %s.]\n\n", shortStandardName(cfg.Standard))
	fmt.Fprintf(sb, "### 10.2 Nonconformity and Corrective Action\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: corrective action process — describe how nonconformities are identified, investigated, corrected, and prevented from recurrence. Reference specimen for POA&M tracking.]\n\n")
}

func writeVersionControl(sb *strings.Builder, cfg DocumentConfig, ctx *ProgramContext) {
	fmt.Fprintf(sb, "---\n\n## Version Control\n\n")
	fmt.Fprintf(sb, "| Version | Date | Author | Changes |\n|---|---|---|---|\n")
	fmt.Fprintf(sb, "| 1.0 | %s | %s | Initial assembly via compound |\n\n", time.Now().Format("2006-01-02"), coalesce(ctx.Owner, "[OWNER NEEDED]"))
	fmt.Fprintf(sb, "*Generated by [compound](https://github.com/Formulary-Labs/compound). %s*\n", standardDisplay(cfg.Standard))
}

//nolint:revive // AssembleDelta intentionally mirrors Assemble for API symmetry.
// AssembleDelta regenerates only the Annex SL clauses whose inputs changed
// between prev and curr ProgramContext. The output is a partial document
// containing only the regenerated clauses plus a version control table that
// records which clauses were updated and why.
//
// When no clause inputs changed, returns an empty string and a note that no
// clauses required regeneration.
func AssembleDelta(cfg DocumentConfig, curr, prev *ProgramContext) string {
	if curr == nil {
		curr = &ProgramContext{Program: cfg.Program}
	}
	if prev == nil {
		prev = &ProgramContext{}
	}

	changed := diffClauses(prev, curr)
	if len(changed) == 0 {
		return fmt.Sprintf("# Delta Review — %s\n\nNo clause inputs changed between the previous and current program state. No clauses require regeneration.\n", cfg.Program)
	}

	orgName := "the Organization"
	if cfg.BrandedOutput && cfg.OrgName != "" {
		orgName = cfg.OrgName
	}

	docName := cfg.OutputName
	if docName == "" {
		docName = standardDocName(cfg.Standard)
	}

	sb := &strings.Builder{}
	fmt.Fprintf(sb, "# Delta Review — %s\n\n", docName)
	fmt.Fprintf(sb, "**Program:** %s  \n**Generated:** %s  \n**Mode:** delta_review — only changed clauses included\n\n---\n\n", curr.Program, time.Now().Format("2006-01-02"))
	fmt.Fprintf(sb, "> This partial document contains only the Annex SL clauses whose inputs changed since the previous run state. Unchanged clauses are preserved verbatim from the prior document.\n\n---\n\n")

	for _, clause := range changed {
		switch clause {
		case 4:
			writeClause4(sb, cfg, curr, orgName)
		case 5:
			writeClause5(sb, cfg, curr, orgName)
		case 6:
			writeClause6(sb, cfg, curr)
		case 7:
			writeClause7(sb, cfg, curr, orgName)
		case 8:
			writeClause8(sb, cfg, curr)
		case 9:
			writeClause9(sb, cfg, curr, orgName)
		case 10:
			writeClause10(sb, cfg, curr)
		}
	}

	// Version control table noting which clauses were updated.
	fmt.Fprintf(sb, "---\n\n## Version Control — Delta\n\n")
	fmt.Fprintf(sb, "| Clause | Updated | Reason |\n|---|---|---|\n")
	for _, clause := range changed {
		fmt.Fprintf(sb, "| Clause %d | %s | Input data changed between run states |\n", clause, time.Now().Format("2006-01-02"))
	}
	fmt.Fprintf(sb, "\n*Generated by [compound](https://github.com/Formulary-Labs/compound) in delta_review mode. %s*\n", standardDisplay(cfg.Standard))
	return sb.String()
}

// AssembleSection regenerates a single Annex SL clause (4–10) targeted by
// cfg.SectionTarget. The output contains only the targeted clause plus a
// version control table. The caller is responsible for merging the output
// back into the existing document.
func AssembleSection(cfg DocumentConfig, ctx *ProgramContext) (string, error) {
	if ctx == nil {
		ctx = &ProgramContext{Program: cfg.Program}
	}

	var clause int
	if _, err := fmt.Sscanf(cfg.SectionTarget, "%d", &clause); err != nil || clause < 4 || clause > 10 {
		return "", fmt.Errorf("--section must be an integer from 4 to 10 (got %q)", cfg.SectionTarget)
	}

	orgName := "the Organization"
	if cfg.BrandedOutput && cfg.OrgName != "" {
		orgName = cfg.OrgName
	}

	docName := cfg.OutputName
	if docName == "" {
		docName = standardDocName(cfg.Standard)
	}

	sb := &strings.Builder{}
	fmt.Fprintf(sb, "# Section Update — %s — Clause %d\n\n", docName, clause)
	fmt.Fprintf(sb, "**Program:** %s  \n**Generated:** %s  \n**Mode:** section_update — Clause %d only\n\n---\n\n", ctx.Program, time.Now().Format("2006-01-02"), clause)
	fmt.Fprintf(sb, "> Replace Clause %d in the existing document with the content below. All other clauses are unchanged.\n\n---\n\n", clause)

	switch clause {
	case 4:
		writeClause4(sb, cfg, ctx, orgName)
	case 5:
		writeClause5(sb, cfg, ctx, orgName)
	case 6:
		writeClause6(sb, cfg, ctx)
	case 7:
		writeClause7(sb, cfg, ctx, orgName)
	case 8:
		writeClause8(sb, cfg, ctx)
	case 9:
		writeClause9(sb, cfg, ctx, orgName)
	case 10:
		writeClause10(sb, cfg, ctx)
	}

	fmt.Fprintf(sb, "---\n\n## Version Control — Section Update\n\n")
	fmt.Fprintf(sb, "| Clause | Updated | Author | Changes |\n|---|---|---|---|\n")
	fmt.Fprintf(sb, "| Clause %d | %s | %s | Section regenerated via compound section_update |\n\n", clause, time.Now().Format("2006-01-02"), coalesce(ctx.Owner, "[OWNER NEEDED]"))
	fmt.Fprintf(sb, "*Generated by [compound](https://github.com/Formulary-Labs/compound) in section_update mode. %s*\n", standardDisplay(cfg.Standard))

	return sb.String(), nil
}

// diffClauses returns the Annex SL clause numbers (4–10) whose inputs changed
// between prev and curr ProgramContext. Only clauses with data-driven content
// from ProgramContext are included; narrative-only clauses (7, 9, 10) are
// included when no other change is detected to ensure the document reflects
// any indirect context shift.
func diffClauses(prev, curr *ProgramContext) []int {
	var changed []int

	// Clause 4 — scope, product, framework.
	if prev.Scope != curr.Scope || prev.ProductName != curr.ProductName || prev.Framework != curr.Framework {
		changed = append(changed, 4)
	}

	// Clause 5 — owner/leadership.
	if prev.Owner != curr.Owner {
		changed = append(changed, 5)
	}

	// Clause 6 — risk posture.
	if prev.RiskCount != curr.RiskCount || prev.OpenRisks != curr.OpenRisks || prev.CriticalRisks != curr.CriticalRisks {
		changed = append(changed, 6)
	}

	// Clause 8 — coverage data.
	prevCov := coverageKey(prev)
	currCov := coverageKey(curr)
	if prevCov != currCov {
		changed = append(changed, 8)
	}

	return changed
}

// coverageKey returns a comparable string key for the coverage block of a
// ProgramContext. Returns "nil" when coverage is absent.
func coverageKey(ctx *ProgramContext) string {
	if ctx == nil || ctx.Coverage == nil {
		return "nil"
	}
	return fmt.Sprintf("%d/%.2f/%.2f", ctx.Coverage.TotalControls, ctx.Coverage.EvidencedPct, ctx.Coverage.GapPct)
}

// --- helpers ---

func standardDocName(s Standard) string {
	switch s {
	case ISO27001:
		return "Information Security Management System (ISMS)"
	case ISO42001:
		return "AI Management System (AIMS)"
	case IEC62443:
		return "Cybersecurity Management System (CSMS)"
	default:
		return "Management System"
	}
}

func shortStandardName(s Standard) string {
	switch s {
	case ISO27001:
		return "ISMS"
	case ISO42001:
		return "AIMS"
	case IEC62443:
		return "CSMS"
	default:
		return "management system"
	}
}

func standardDisplay(s Standard) string {
	switch s {
	case ISO27001:
		return "ISO/IEC 27001:2022"
	case ISO42001:
		return "ISO/IEC 42001:2023"
	case IEC62443:
		return "IEC 62443"
	default:
		return string(s)
	}
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
