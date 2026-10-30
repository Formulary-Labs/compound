# compound

`compound` builds Annex SL-structured management system documents. It fills every section it can derive from data — either a program run-state or OSCAL/gemara artifacts — and leaves prose gaps marked for the agent layer.

```bash
go get github.com/Formulary-Labs/compound
```

## In the ecosystem

Formulary is a **pick-and-choose** suite of small, single-purpose CLIs. Install only the tools your workflow needs; each runs standalone. Shared contracts live in [`substrate`](https://github.com/Formulary-Labs/substrate). Optional Make pipelines (or an agent such as `regimen`) compose tools into a run — plug-and-play, not all-or-nothing.

**This tool:** Annex SL security plan assembly (`--assemble-plan`) with hard packet boundary.  
**Works with:** `distill` / `probe` upstream; `formula` / `specimen` for cited packet paths.  
**Pipeline stage:** assemble.


## Modes

```sh
# Assemble with scope and org branding
compound --assemble-plan --program iso27001 --standard iso27001 \
  --oscal data/iso27001/oscal/ --output docs/ISMS-security-plan.md \
  --scope "Production environment — SaaS platform" --org "Example Corp"

# Map policy IDs to clauses when artifacts lack applicability-groups
compound --assemble-plan --program iso27001 --standard iso27001 \
  --oscal data/iso27001/oscal/ --clause-map clause-map.json

# Dry run — print what would be written
compound --assemble-plan --program iso27001 --standard iso27001 \
  --oscal data/iso27001/oscal/ --dry-run
```

Other flags: `--scope` (scope statement), `--org` (organization name for branded output), `--clause-map` (policy-ID → clause map, JSON or simple YAML), `--dry-run`, `--format md|json` (JSON summary only).

### Run-state assembly (classic)

```sh
compound --program iso42001 --standard iso42001 --output docs/AIMS.md
```

Reads `ProgramContext` from run state JSON (coverage, risk counts, scope). Sections needing judgment are flagged `[DATA NEEDED: narrative]`.

### Assemble-plan (security plan)

```sh
compound --assemble-plan \
  --program iso27001 --standard iso27001 \
  --oscal data/iso27001/oscal/ \
  --output docs/ISMS-security-plan.md \
  --report docs/assembly-report.json
```

`--gemara` is a deprecated alias for `--oscal`.

Assembles a **security plan** (ISMS / AIMS / CSMS) from OSCAL catalogs and gemara layers:

| Input | Role |
|---|---|
| gemara `Policy` | PURPOSE / SCOPE / POLICY DETAILS (verbatim `metadata.description`), RACI, clause-bound via `applicability-groups` |
| OSCAL `catalog` or gemara `ControlCatalog` | Clause 8 control themes / framework identity |
| gemara `AuditLog` | Clause 9.2 citations |
| gemara `RiskCatalog` | **Cited only** — never inlined as grids |

OSCAL system security plans, assessment results, and POA&Ms validate under `probe` but are not projected into clause prose in this release.

Hard packet boundary: SoA, risk registers, and impact worksheets stay in the assessment packet (`formula` / `specimen`). The plan references them; it does not paste them.

For auditor narratives that *do* include risk/eval inline, use [`appraise`](https://github.com/Formulary-Labs/appraise).

## Review modes

| Mode | What it generates |
|---|---|
| `full_assembly` | Complete Clauses 4–10 document |
| `delta_review` | Only sections with changes (run-state or artifact fingerprint) |
| `section_update` | A single clause 4–10 (run-state path) |

```sh
# Artifact delta. --prior-gemara remains accepted as an alias of --prior-oscal.
compound --assemble-plan --program iso27001 --standard iso27001 \
  --oscal data/iso27001/oscal/ --review-mode delta_review \
  --prior-oscal data/iso27001/oscal-prior/
```

## Library usage

```go
import "github.com/Formulary-Labs/compound/assemble"

set, err := assemble.LoadArtifactSet([]string{"data/iso27001/oscal"}, "")
result := assemble.AssemblePlan(assemble.DocumentConfig{
    Program:  "iso27001",
    Standard: assemble.ISO27001,
}, set)
// result.Markdown — security plan
// result.Report  — clause ← artifact provenance
```

## Standards

| Constant | Standard |
|---|---|
| `assemble.ISO27001` | ISO/IEC 27001:2022 — ISMS |
| `assemble.ISO42001` | ISO/IEC 42001:2023 — AIMS |
| `assemble.IEC62443` | IEC 62443 — CSMS |

## Pipeline

```text
distill → probe → compound --assemble-plan → challenge → exhibit
                 ↘ formula / titer / specimen  (packet, cite-only)
```

See root `make pipeline-assemble-plan`.

## License

Apache License 2.0
