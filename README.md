# compound

`compound` builds Annex SL-structured management system documents. It fills every section it can derive from data — either a program run-state or gemara artifacts — and leaves prose gaps marked for the agent layer.

```bash
go get github.com/Formulary-Labs/compound
```

## Modes

### Run-state assembly (classic)

```sh
compound --program iso42001 --standard iso42001 --output docs/AIMS.md
```

Reads `ProgramContext` from run state JSON (coverage, risk counts, scope). Sections needing judgment are flagged `[DATA NEEDED: narrative]`.

### Gemara assemble-plan (security plan)

```sh
compound --assemble-plan \
  --program iso27001 --standard iso27001 \
  --gemara data/iso27001/gemara/ \
  --output docs/ISMS-security-plan.md \
  --report docs/assembly-report.json
```

Assembles a **security plan** (ISMS / AIMS / CSMS) from gemara layers:

| Input | Role |
|---|---|
| `Policy` | PURPOSE / SCOPE / POLICY DETAILS (verbatim `metadata.description`), RACI, clause-bound via `applicability-groups` |
| `ControlCatalog` | Clause 8 control themes / framework identity |
| `AuditLog` | Clause 9.2 citations |
| `RiskCatalog` | **Cited only** — never inlined as grids |

Hard packet boundary: SoA, risk registers, and impact worksheets stay in the assessment packet (`formula` / `specimen`). The plan references them; it does not paste them.

For auditor narratives that *do* include risk/eval inline, use [`appraise`](https://github.com/Formulary-Labs/appraise).

## Review modes

| Mode | What it generates |
|---|---|
| `full_assembly` | Complete Clauses 4–10 document |
| `delta_review` | Only sections with changes (run-state or gemara fingerprint) |
| `section_update` | A single clause 4–10 (run-state path) |

```sh
# Gemara delta
compound --assemble-plan --program iso27001 --standard iso27001 \
  --gemara data/iso27001/gemara/ --review-mode delta_review \
  --prior-gemara data/iso27001/gemara-prior/
```

## Library usage

```go
import "github.com/Formulary-Labs/compound/assemble"

set, err := assemble.LoadGemaraSet([]string{"data/iso27001/gemara"}, "")
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
