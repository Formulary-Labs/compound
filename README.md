# compound

`compound` builds the Annex SL structure and fills every section it can derive from data. The sections requiring prose — scope statements, leadership commitment, management review narratives — it leaves marked for you.

```bash
go get github.com/Formulary-Labs/compound
```

## What it does

`compound` produces Annex SL-structured management system documents (Clauses 4–10) from program artifacts. It fills every section where the content can be derived from the program's coverage, risk, and scope data. Sections that require prose are flagged `[DATA NEEDED: narrative — agent layer]`.

Structure and data from `compound`. Prose from the agent layer. The division is explicit and auditable.

Supported standards: ISO/IEC 27001:2022 (ISMS), ISO/IEC 42001:2023 (AIMS), IEC 62443 (CSMS).

## Usage

```go
import "github.com/Formulary-Labs/compound/assemble"

doc, err := assemble.Build(assemble.DocumentConfig{
    Program:       "my-program",
    Standard:      assemble.ISO27001,
    OutputName:    "isms-2026",
    Applicability: "Production SaaS platform — EU region",
    ReviewCadence: "annual",
    ReviewMode:    assemble.FullAssembly,
    OrgName:       "Acme Corp",
})

// doc is a Markdown string — write it, render it, or pass it to an agent layer
```

## Standards

| Constant | Standard |
|---|---|
| `assemble.ISO27001` | ISO/IEC 27001:2022 — Information Security Management System |
| `assemble.ISO42001` | ISO/IEC 42001:2023 — AI Management System |
| `assemble.IEC62443` | IEC 62443 — Cybersecurity Management System |

## Review modes

| Mode | What it generates |
|---|---|
| `FullAssembly` | Complete Clauses 4–10 document |
| `DeltaReview` | Only sections with changes since the last version |
| `SectionUpdate` | A single target section |

```go
// Regenerate only Clause 6 — Planning
assemble.DocumentConfig{
    ReviewMode:    assemble.SectionUpdate,
    SectionTarget: "6",
}
```

`DeltaReview` and `SectionUpdate` modes require a prior version of the document to diff against. Pass the prior document's path in `DocumentConfig.PriorVersion`.

## ProgramContext input

`compound` reads program data assembled from `titer`, `specimen`, and the program's run state:

```go
assemble.ProgramContext{
    Program:       "my-program",
    Standard:      "iso27001",
    Scope:         "Production SaaS platform — EU region",
    ProductName:   "My Product",
    Coverage: &struct {
        TotalControls int
        EvidencedPct  float64
        GapPct        float64
    }{TotalControls: 114, EvidencedPct: 82.0, GapPct: 12.0},
    RiskCount:     14,
    OpenRisks:     6,
    CriticalRisks: 0,
    Owner:         "security-team",
}
```

## Output structure

The output is a single Markdown string with:

- Version control table (version, date, author, change summary)
- Clauses 4–10 with section headings matching the standard's clause structure
- Data-filled sections: scope, coverage metrics, risk summary, control family applicability
- `[DATA NEEDED: narrative — agent layer]` for all prose sections requiring judgment

Pass the output to the agent layer to fill `[DATA NEEDED]` placeholders, then review before submission.

## Working with [DATA NEEDED] placeholders

Each placeholder identifies the specific section and the type of content required:

```markdown
## Clause 5.1 — Leadership and Commitment

[DATA NEEDED: narrative — agent layer]
Describe how top management demonstrates leadership and commitment
to the information security management system.
```

The agent layer (regimen) fills these using the program's constitution, memory, decisions log, and context. `compound` makes its scope boundary explicit so the generated sections are distinguishable from authored sections in the review process.

## License

Apache License 2.0
