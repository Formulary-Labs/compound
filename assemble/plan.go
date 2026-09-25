package assemble

import (
	"fmt"
	"strings"
	"time"

	"github.com/Formulary-Labs/substrate/artifact"
)

// AssemblyReport records which gemara artifacts filled which clause slots.
type AssemblyReport struct {
	Program           string             `json:"program"`
	Standard          string             `json:"standard"`
	Generated         string             `json:"generated"`
	GemaraFingerprint string             `json:"gemara_fingerprint"`
	Clauses           []ClauseProvenance `json:"clauses"`
	DataNeededCount   int                `json:"data_needed_count"`
	CitationsOnly     []CitationRef      `json:"citations_only,omitempty"`
}

// ClauseProvenance maps a clause slot to its gemara sources.
type ClauseProvenance struct {
	Clause      string   `json:"clause"`
	Sources     []string `json:"sources"`
	Filled      bool     `json:"filled"`
	Placeholder string   `json:"placeholder,omitempty"`
}

// CitationRef is a packet/risk artifact cited but not inlined.
type CitationRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Path string `json:"path"`
	Note string `json:"note"`
}

// PlanResult is the assembled security plan markdown plus provenance report.
type PlanResult struct {
	Markdown string
	Report   AssemblyReport
}

// AssemblePlan builds a security plan from gemara artifacts with a hard packet
// boundary: SoA / risk grids / impact worksheets are cited by reference only.
func AssemblePlan(cfg DocumentConfig, set *GemaraSet) PlanResult {
	if set == nil {
		set = &GemaraSet{}
	}
	if string(cfg.Standard) == "" {
		cfg.Standard = inferStandard(set)
	}

	genAt := cfg.GeneratedAt
	if genAt.IsZero() {
		genAt = time.Now().UTC()
	}
	genDate := genAt.Format("2006-01-02")

	docName := cfg.OutputName
	if docName == "" {
		docName = securityPlanName(cfg.Standard)
	}
	orgName := "the Organization"
	if cfg.BrandedOutput && cfg.OrgName != "" {
		orgName = cfg.OrgName
	}
	program := cfg.Program
	if program == "" {
		program = "program"
	}

	report := AssemblyReport{
		Program:           program,
		Standard:          string(cfg.Standard),
		Generated:         genDate,
		GemaraFingerprint: set.SetFingerprint(),
	}
	for _, rc := range set.RiskCatalogs {
		id := ""
		if rc.Catalog != nil {
			id = rc.Catalog.Metadata.Id
		}
		report.CitationsOnly = append(report.CitationsOnly, CitationRef{
			Type: "RiskCatalog",
			ID:   id,
			Path: rc.Path,
			Note: "assessment packet — cited only, not inlined into plan body",
		})
	}

	sb := &strings.Builder{}
	writePlanHeader(sb, docName, cfg, program, orgName, genDate)

	report.Clauses = append(report.Clauses,
		writePlanClause4(sb, cfg, set, orgName),
		writePlanClause5(sb, cfg, set),
		writePlanClause6(sb, cfg, set),
		writePlanClause7(sb, cfg, set, orgName),
		writePlanClause8(sb, cfg, set, program),
		writePlanClause9(sb, set),
		writePlanClause10(sb, cfg),
	)

	writePlanVersionControl(sb, cfg, set, genDate)
	writePlanProvenanceFooter(sb, report)

	md := sb.String()
	report.DataNeededCount = strings.Count(md, "[DATA NEEDED")
	return PlanResult{Markdown: md, Report: report}
}

// AssemblePlanDelta regenerates when gemara sources changed between prev and curr.
func AssemblePlanDelta(cfg DocumentConfig, curr, prev *GemaraSet) PlanResult {
	if curr == nil {
		curr = &GemaraSet{}
	}
	if prev == nil {
		prev = &GemaraSet{}
	}
	if curr.SetFingerprint() == prev.SetFingerprint() {
		note := fmt.Sprintf("# Delta Review — %s\n\nNo gemara inputs changed between the previous and current artifact set. No clauses require regeneration.\n", cfg.Program)
		return PlanResult{
			Markdown: note,
			Report: AssemblyReport{
				Program:           cfg.Program,
				Standard:          string(cfg.Standard),
				GemaraFingerprint: curr.SetFingerprint(),
			},
		}
	}
	return AssemblePlan(cfg, curr)
}

func writePlanHeader(sb *strings.Builder, docName string, cfg DocumentConfig, program, orgName, genDate string) {
	fmt.Fprintf(sb, "# %s\n\n", docName)
	fmt.Fprintf(sb, "**Standard:** %s  \n", standardDisplay(cfg.Standard))
	fmt.Fprintf(sb, "**Program:** %s  \n", program)
	fmt.Fprintf(sb, "**Organization:** %s  \n", orgName)
	if cfg.Applicability != "" {
		fmt.Fprintf(sb, "**Scope:** %s  \n", cfg.Applicability)
	}
	cadence := cfg.ReviewCadence
	if cadence == "" {
		cadence = "annual"
	}
	fmt.Fprintf(sb, "**Review Cadence:** %s  \n", cadence)
	fmt.Fprintf(sb, "**Generated:** %s\n\n", genDate)
	fmt.Fprintf(sb, "---\n\n")
	fmt.Fprintf(sb, "> Assembled by [compound](https://github.com/Formulary-Labs/compound) `--assemble-plan` from gemara artifacts. Assessment-packet materials (SoA, risk registers, impact worksheets) are cited by reference only. Sections marked `[DATA NEEDED: gemara …]` lack a source artifact.\n\n")
	fmt.Fprintf(sb, "---\n\n")
}

func writePlanClause4(sb *strings.Builder, cfg DocumentConfig, set *GemaraSet, orgName string) ClauseProvenance {
	fmt.Fprintf(sb, "## Clause 4 — Context of the Organization\n\n")
	policies := set.PoliciesForClause("4")
	sources := policySources(policies)
	filled := false

	fmt.Fprintf(sb, "### 4.1 Understanding the Organization and Its Context\n\n")
	if desc := firstPolicyDescription(policies); desc != "" {
		fmt.Fprintf(sb, "%s\n\n", desc)
		filled = true
	} else if set.Catalog != nil && set.Catalog.Metadata.Description != "" {
		fmt.Fprintf(sb, "%s\n\n", set.Catalog.Metadata.Description)
		sources = append(sources, catalogSource(set))
		filled = true
	} else {
		fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/4 — describe internal and external context for the %s.]\n\n", shortStandardName(cfg.Standard))
	}

	fmt.Fprintf(sb, "### 4.2 Understanding the Needs and Expectations of Interested Parties\n\n")
	if p := firstPolicyWithContacts(policies); p != nil {
		writeInterestedParties(sb, p)
		filled = true
	} else {
		fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/4 — list interested parties and their requirements.]\n\n")
	}

	fmt.Fprintf(sb, "### 4.3 Determining the Scope of the %s\n\n", shortStandardName(cfg.Standard))
	if cfg.Applicability != "" {
		fmt.Fprintf(sb, "**Scope statement:** %s\n\n", cfg.Applicability)
		filled = true
	} else if p := firstPolicyWithScope(policies); p != nil {
		writeScopeBlock(sb, p)
		filled = true
	} else {
		fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/4 — provide scope boundary (in/out).]\n\n")
	}

	fmt.Fprintf(sb, "### 4.4 %s\n\n", shortStandardName(cfg.Standard))
	fmt.Fprintf(sb, "%s has established, implemented, maintains, and continually improves a %s in accordance with %s.\n\n",
		orgName, standardDocName(cfg.Standard), standardDisplay(cfg.Standard))

	return clauseProv("4", sources, filled, "")
}

func writePlanClause5(sb *strings.Builder, cfg DocumentConfig, set *GemaraSet) ClauseProvenance {
	fmt.Fprintf(sb, "## Clause 5 — Leadership\n\n")
	fmt.Fprintf(sb, "### 5.1 Leadership and Commitment\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/5.1 — describe how top management demonstrates leadership and commitment.]\n\n")

	fmt.Fprintf(sb, "### 5.2 Policy\n\n")
	policies := set.PoliciesForClause("5.2")
	sources := policySources(policies)
	filled := false

	if len(policies) == 0 {
		fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/5.2 — provide PURPOSE / SCOPE / POLICY DETAILS.]\n\n")
	} else {
		for _, pref := range policies {
			p := pref.Policy
			fmt.Fprintf(sb, "#### %s\n\n", coalesce(p.Title, "Policy"))
			fmt.Fprintf(sb, "*Source: `%s` (Policy `%s`)*\n\n", pref.Path, p.Metadata.Id)

			fmt.Fprintf(sb, "**PURPOSE**\n\n")
			if p.Metadata.Description != "" {
				fmt.Fprintf(sb, "%s\n\n", p.Metadata.Description)
				filled = true
			} else {
				fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/%s description — PURPOSE text.]\n\n", p.Metadata.Id)
			}

			fmt.Fprintf(sb, "**SCOPE**\n\n")
			writeScopeBlock(sb, p)
			if policyHasScope(p) {
				filled = true
			}

			fmt.Fprintf(sb, "**POLICY DETAILS**\n\n")
			details := policyDetails(p)
			if details != "" {
				fmt.Fprintf(sb, "%s\n\n", details)
				filled = true
			} else {
				fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/%s — POLICY DETAILS (adherence / implementation notes).]\n\n", p.Metadata.Id)
			}
		}
	}

	fmt.Fprintf(sb, "### 5.3 Organizational Roles, Responsibilities, and Authorities\n\n")
	rolePolicies := set.PoliciesForClause("5.3")
	if len(rolePolicies) == 0 {
		rolePolicies = set.PoliciesForClause("5.2")
	}
	roleFilled := false
	if p := firstPolicyWithContacts(rolePolicies); p != nil {
		writeRACI(sb, p)
		sources = append(sources, policySources(rolePolicies)...)
		roleFilled = true
	} else {
		fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/5.3 — RACI contacts for %s management.]\n\n", shortStandardName(cfg.Standard))
	}

	return clauseProv("5", sources, filled || roleFilled, "")
}

func writePlanClause6(sb *strings.Builder, cfg DocumentConfig, set *GemaraSet) ClauseProvenance {
	fmt.Fprintf(sb, "## Clause 6 — Planning\n\n")
	fmt.Fprintf(sb, "### 6.1 Actions to Address Risks and Opportunities\n\n")
	sources := []string{}
	filled := false

	// Hard packet boundary: cite RiskCatalog only — never dump risk grids or counts.
	if len(set.RiskCatalogs) > 0 {
		fmt.Fprintf(sb, "Risk identification and treatment records are maintained in the assessment packet:\n\n")
		for _, rc := range set.RiskCatalogs {
			id := rc.Path
			if rc.Catalog != nil && rc.Catalog.Metadata.Id != "" {
				id = rc.Catalog.Metadata.Id
			}
			fmt.Fprintf(sb, "- RiskCatalog `%s` (`%s`) — see assessment packet; not reproduced in this plan.\n", id, rc.Path)
			sources = append(sources, fmt.Sprintf("RiskCatalog:%s (%s)", id, rc.Path))
		}
		fmt.Fprintf(sb, "\n")
		filled = true
	}
	fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/6.1 — describe the process for identifying, assessing, and treating risks (process text only; risk register remains packet).]\n\n")

	fmt.Fprintf(sb, "### 6.2 %s Objectives and Planning to Achieve Them\n\n", shortStandardName(cfg.Standard))
	fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/6.2 — list %s objectives with owners, measures, and target dates.]\n\n", shortStandardName(cfg.Standard))

	if cfg.Standard == ISO27001 || cfg.Standard == ISO42001 {
		fmt.Fprintf(sb, "### 6.3 Planning of Changes\n\n")
		fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/6.3 — describe how planned changes to the %s are managed.]\n\n", shortStandardName(cfg.Standard))
	}

	return clauseProv("6", sources, filled, "")
}

func writePlanClause7(sb *strings.Builder, cfg DocumentConfig, set *GemaraSet, orgName string) ClauseProvenance {
	fmt.Fprintf(sb, "## Clause 7 — Support\n\n")
	policies := set.PoliciesForClause("7.4")
	sources := policySources(policies)
	filled := false

	fmt.Fprintf(sb, "### 7.1 Resources\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/7.1 — describe resources supporting the %s.]\n\n", shortStandardName(cfg.Standard))
	fmt.Fprintf(sb, "### 7.2 Competence\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/7.2 — describe competence requirements and maintenance.]\n\n")
	fmt.Fprintf(sb, "### 7.3 Awareness\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/7.3 — describe the awareness program.]\n\n")
	fmt.Fprintf(sb, "### 7.4 Communication\n\n")
	if desc := firstPolicyDescription(policies); desc != "" {
		fmt.Fprintf(sb, "%s\n\n", desc)
		filled = true
	} else {
		fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/7.4 — communication plan (what, when, whom, channels).]\n\n")
	}
	fmt.Fprintf(sb, "### 7.5 Documented Information\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/7.5 — document inventory required by %s and maintained by %s.]\n\n",
		standardDisplay(cfg.Standard), orgName)

	return clauseProv("7", sources, filled, "")
}

func writePlanClause8(sb *strings.Builder, cfg DocumentConfig, set *GemaraSet, program string) ClauseProvenance {
	fmt.Fprintf(sb, "## Clause 8 — Operation\n\n")
	fmt.Fprintf(sb, "### 8.1 Operational Planning and Control\n\n")
	sources := []string{}
	filled := false

	if set.Catalog != nil {
		fmt.Fprintf(sb, "Control themes are defined in ControlCatalog `%s` (`%s`):\n\n", set.Catalog.Metadata.Id, set.CatalogPath)
		sources = append(sources, catalogSource(set))
		if len(set.Catalog.Groups) > 0 {
			fmt.Fprintf(sb, "| Family ID | Title |\n|---|---|\n")
			for _, g := range set.Catalog.Groups {
				fmt.Fprintf(sb, "| %s | %s |\n", g.Id, g.Title)
			}
			fmt.Fprintf(sb, "\n")
			filled = true
		} else if len(set.Catalog.Controls) > 0 {
			fmt.Fprintf(sb, "Catalog defines **%d controls**.\n\n", len(set.Catalog.Controls))
			filled = true
		}
	} else {
		fmt.Fprintf(sb, "[DATA NEEDED: gemara ControlCatalog — control themes for operational planning.]\n\n")
	}

	switch cfg.Standard {
	case ISO42001:
		fmt.Fprintf(sb, "### 8.2 AI Risk Assessment\n\n")
		fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/8.2 — AI risk assessment process (summary only; detailed risks remain in assessment packet).]\n\n")
		fmt.Fprintf(sb, "### 8.3 AI Risk Treatment\n\n")
		fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/8.3 — AI risk treatment process.]\n\n")
		fmt.Fprintf(sb, "### 8.4 Statement of Applicability\n\n")
		fmt.Fprintf(sb, "The Statement of Applicability is an **assessment-packet** artifact (generated via formula). It is not reproduced in this security plan. Current packet path: `data/%s/soa.csv`.\n\n", program)
	case ISO27001:
		fmt.Fprintf(sb, "### 8.2 Information Security Risk Assessment\n\n")
		fmt.Fprintf(sb, "Detailed risk assessment outputs remain in the assessment packet")
		if len(set.RiskCatalogs) > 0 {
			fmt.Fprintf(sb, " (see RiskCatalog citations in Clause 6)")
		}
		fmt.Fprintf(sb, ".\n\n")
		fmt.Fprintf(sb, "### 8.3 Information Security Risk Treatment\n\n")
		fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/8.3 — risk treatment process description.]\n\n")
	}

	return clauseProv("8", sources, filled, "")
}

func writePlanClause9(sb *strings.Builder, set *GemaraSet) ClauseProvenance {
	fmt.Fprintf(sb, "## Clause 9 — Performance Evaluation\n\n")
	sources := []string{}
	filled := false

	fmt.Fprintf(sb, "### 9.1 Monitoring, Measurement, Analysis, and Evaluation\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/9.1 — KPI/KRI monitoring process. Coverage metrics (titer) and risk posture (specimen) remain packet/citation.]\n\n")

	fmt.Fprintf(sb, "### 9.2 Internal Audit\n\n")
	if len(set.AuditLogs) > 0 {
		fmt.Fprintf(sb, "Internal audit activity is recorded in gemara AuditLog artifacts:\n\n")
		for _, a := range set.AuditLogs {
			id := a.Path
			summary := ""
			if a.Log != nil {
				id = a.Log.Metadata.Id
				summary = a.Log.Summary
			}
			fmt.Fprintf(sb, "- AuditLog `%s` (`%s`)", id, a.Path)
			if summary != "" {
				fmt.Fprintf(sb, " — %s", summary)
			}
			fmt.Fprintf(sb, "\n")
			sources = append(sources, fmt.Sprintf("AuditLog:%s (%s)", id, a.Path))
		}
		fmt.Fprintf(sb, "\n")
		filled = true
	} else {
		fmt.Fprintf(sb, "[DATA NEEDED: gemara AuditLog/9.2 — internal audit program schedule, scope, and reporting.]\n\n")
	}

	fmt.Fprintf(sb, "### 9.3 Management Review\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/9.3 — management review process, frequency, inputs, and outputs.]\n\n")

	return clauseProv("9", sources, filled, "")
}

func writePlanClause10(sb *strings.Builder, cfg DocumentConfig) ClauseProvenance {
	fmt.Fprintf(sb, "## Clause 10 — Improvement\n\n")
	fmt.Fprintf(sb, "### 10.1 Continual Improvement\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/10.1 — continual improvement process for the %s.]\n\n", shortStandardName(cfg.Standard))
	fmt.Fprintf(sb, "### 10.2 Nonconformity and Corrective Action\n\n")
	fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy/10.2 — nonconformity and corrective action process. POA&M tracking remains assessment packet.]\n\n")
	return clauseProv("10", nil, false, "[DATA NEEDED: gemara Policy/10]")
}

func writePlanVersionControl(sb *strings.Builder, cfg DocumentConfig, set *GemaraSet, genDate string) {
	fmt.Fprintf(sb, "---\n\n## Version Control\n\n")
	fmt.Fprintf(sb, "| Version | Date | Author | Changes |\n|---|---|---|\n")
	author := "[OWNER NEEDED]"
	if p := primaryPolicy(set); p != nil && p.Policy.Metadata.Author.Name != "" {
		author = p.Policy.Metadata.Author.Name
	}
	fmt.Fprintf(sb, "| 1.0 | %s | %s | Initial assemble-plan via compound |\n\n", genDate, author)
	fmt.Fprintf(sb, "*Generated by [compound](https://github.com/Formulary-Labs/compound) assemble-plan. %s*\n", standardDisplay(cfg.Standard))
}

func writePlanProvenanceFooter(sb *strings.Builder, report AssemblyReport) {
	fmt.Fprintf(sb, "\n---\n\n## Assembly Provenance\n\n")
	fmt.Fprintf(sb, "| Clause | Filled | Sources |\n|---|---|---|\n")
	for _, c := range report.Clauses {
		src := "—"
		if len(c.Sources) > 0 {
			src = strings.Join(c.Sources, "; ")
		}
		fmt.Fprintf(sb, "| %s | %v | %s |\n", c.Clause, c.Filled, src)
	}
	fmt.Fprintf(sb, "\n")
}

func securityPlanName(s Standard) string {
	switch s {
	case ISO27001:
		return "Information Security Management System (ISMS) — Security Plan"
	case ISO42001:
		return "AI Management System (AIMS) — Security Plan"
	case IEC62443:
		return "Cybersecurity Management System (CSMS) — Security Plan"
	default:
		return "Security Plan"
	}
}

func inferStandard(set *GemaraSet) Standard {
	if set == nil || set.Catalog == nil {
		return ""
	}
	blob := strings.ToLower(set.Catalog.Title + " " + set.Catalog.Metadata.Description + " " + set.Catalog.Metadata.Id)
	switch {
	case strings.Contains(blob, "42001"):
		return ISO42001
	case strings.Contains(blob, "62443"):
		return IEC62443
	case strings.Contains(blob, "27001"):
		return ISO27001
	default:
		return ""
	}
}

func clauseProv(clause string, sources []string, filled bool, placeholder string) ClauseProvenance {
	return ClauseProvenance{
		Clause:      clause,
		Sources:     uniqueStrings(sources),
		Filled:      filled,
		Placeholder: placeholder,
	}
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func policySources(policies []PolicyRef) []string {
	var out []string
	for _, p := range policies {
		id := p.Path
		if p.Policy != nil && p.Policy.Metadata.Id != "" {
			id = p.Policy.Metadata.Id
		}
		out = append(out, fmt.Sprintf("Policy:%s (%s)", id, p.Path))
	}
	return out
}

func catalogSource(set *GemaraSet) string {
	if set == nil || set.Catalog == nil {
		return ""
	}
	return fmt.Sprintf("ControlCatalog:%s (%s)", set.Catalog.Metadata.Id, set.CatalogPath)
}

func primaryPolicy(set *GemaraSet) *PolicyRef {
	if set == nil || len(set.Policies) == 0 {
		return nil
	}
	if ps := set.PoliciesForClause("5.2"); len(ps) > 0 {
		return &ps[0]
	}
	return &set.Policies[0]
}

func firstPolicyDescription(policies []PolicyRef) string {
	for _, p := range policies {
		if p.Policy != nil && p.Policy.Metadata.Description != "" {
			return p.Policy.Metadata.Description
		}
	}
	return ""
}

func firstPolicyWithScope(policies []PolicyRef) *artifact.Policy {
	for _, p := range policies {
		if p.Policy != nil && policyHasScope(p.Policy) {
			return p.Policy
		}
	}
	return nil
}

func firstPolicyWithContacts(policies []PolicyRef) *artifact.Policy {
	for _, p := range policies {
		if p.Policy != nil && policyHasContacts(p.Policy) {
			return p.Policy
		}
	}
	return nil
}

func policyHasScope(p *artifact.Policy) bool {
	if p == nil {
		return false
	}
	s := p.Scope
	return len(s.In.Technologies) > 0 || len(s.In.Geopolitical) > 0 || len(s.In.Sensitivity) > 0 ||
		len(s.In.Users) > 0 || len(s.In.Groups) > 0 ||
		len(s.Out.Technologies) > 0 || len(s.Out.Geopolitical) > 0
}

func policyHasContacts(p *artifact.Policy) bool {
	if p == nil {
		return false
	}
	c := p.Contacts
	return len(c.Responsible) > 0 || len(c.Accountable) > 0 || len(c.Consulted) > 0 || len(c.Informed) > 0
}

func writeScopeBlock(sb *strings.Builder, p *artifact.Policy) {
	if p == nil || !policyHasScope(p) {
		fmt.Fprintf(sb, "[DATA NEEDED: gemara Policy scope — in/out dimensions.]\n\n")
		return
	}
	s := p.Scope
	if len(s.In.Technologies) > 0 {
		fmt.Fprintf(sb, "**In-scope technologies:** %s\n\n", strings.Join(s.In.Technologies, ", "))
	}
	if len(s.In.Geopolitical) > 0 {
		fmt.Fprintf(sb, "**In-scope geopolitical:** %s\n\n", strings.Join(s.In.Geopolitical, ", "))
	}
	if len(s.In.Sensitivity) > 0 {
		fmt.Fprintf(sb, "**In-scope sensitivity:** %s\n\n", strings.Join(s.In.Sensitivity, ", "))
	}
	if len(s.Out.Technologies) > 0 {
		fmt.Fprintf(sb, "**Out-of-scope technologies:** %s\n\n", strings.Join(s.Out.Technologies, ", "))
	}
	if len(s.Out.Geopolitical) > 0 {
		fmt.Fprintf(sb, "**Out-of-scope geopolitical:** %s\n\n", strings.Join(s.Out.Geopolitical, ", "))
	}
}

func writeRACI(sb *strings.Builder, p *artifact.Policy) {
	if p == nil {
		return
	}
	c := p.Contacts
	fmt.Fprintf(sb, "| Role | Name | Affiliation |\n|---|---|---|\n")
	writeContactRows(sb, "Responsible", c.Responsible)
	writeContactRows(sb, "Accountable", c.Accountable)
	writeContactRows(sb, "Consulted", c.Consulted)
	writeContactRows(sb, "Informed", c.Informed)
	fmt.Fprintf(sb, "\n")
}

func writeInterestedParties(sb *strings.Builder, p *artifact.Policy) {
	fmt.Fprintf(sb, "Interested parties derived from Policy contacts:\n\n")
	writeRACI(sb, p)
}

func writeContactRows(sb *strings.Builder, role string, contacts []artifact.Contact) {
	for _, ct := range contacts {
		aff := ""
		if ct.Affiliation != nil {
			aff = *ct.Affiliation
		}
		fmt.Fprintf(sb, "| %s | %s | %s |\n", role, ct.Name, aff)
	}
}

func policyDetails(p *artifact.Policy) string {
	if p == nil {
		return ""
	}
	var parts []string
	if p.Adherence.NonCompliance != "" {
		parts = append(parts, "**Non-compliance:** "+p.Adherence.NonCompliance)
	}
	if p.ImplementationPlan.NotificationProcess != "" {
		parts = append(parts, "**Notification process:** "+p.ImplementationPlan.NotificationProcess)
	}
	for _, m := range p.Adherence.EvaluationMethods {
		if m.Description != "" {
			parts = append(parts, "- Evaluation: "+m.Description)
		}
	}
	for _, m := range p.Adherence.EnforcementMethods {
		if m.Description != "" {
			parts = append(parts, "- Enforcement: "+m.Description)
		}
	}
	return strings.Join(parts, "\n")
}

