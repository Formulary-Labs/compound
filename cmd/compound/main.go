// compound assembles Annex SL-structured management system documents
// (ISMS, AIMS, CSMS) from program artifacts.
//
// Usage:
//
//	compound --program <slug> --standard <iso27001|iso42001|iec62443> [flags]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Formulary-Labs/compound/assemble"
	"github.com/Formulary-Labs/substrate/exit"
	"github.com/Formulary-Labs/substrate/provenance"
)

const version = "0.1.0"

func main() {
	var (
		programFlag          = flag.String("program", "", "Program slug (required)")
		standardFlag         = flag.String("standard", "", "Standard: iso27001, iso42001, iec62443 (required)")
		runStateFlag         = flag.String("run-state", "", "Path to run state JSON (default: runs/[program]/latest.json)")
		previousRunStateFlag = flag.String("previous-run-state", "", "Path to previous run state JSON (required for delta_review mode)")
		outputFlag           = flag.String("output", "", "Output markdown path (default: stdout)")
		fmtFlag              = flag.String("format", "md", "Output format: md (default), json (JSON summary only)")
		scopeFlag            = flag.String("scope", "", "Scope statement for the management system")
		reviewModeFlag       = flag.String("review-mode", "full_assembly", "Review mode: full_assembly, delta_review, section_update")
		orgNameFlag          = flag.String("org", "", "Organization name for branded output")
		sectionFlag          = flag.String("section", "", "Target clause for section_update mode (4–10, e.g. '8')")
		dryRunFlag           = flag.Bool("dry-run", false, "Print what would be written without writing")
		versionFlag          = flag.Bool("version", false, "Print version and exit")
	)
	flag.Usage = usage
	flag.Parse()

	if *versionFlag {
		fmt.Printf("compound version %s\n", version)
		os.Exit(exit.OK)
	}

	if *programFlag == "" {
		fmt.Fprintln(os.Stderr, `{"error": "--program is required", "code": 2}`)
		flag.Usage()
		os.Exit(exit.ToolError)
	}

	cfg := assemble.DocumentConfig{
		Program:       *programFlag,
		Standard:      assemble.Standard(*standardFlag),
		Applicability: *scopeFlag,
		ReviewMode:    assemble.ReviewMode(*reviewModeFlag),
		SectionTarget: *sectionFlag,
	}
	if *orgNameFlag != "" {
		cfg.BrandedOutput = true
		cfg.OrgName = *orgNameFlag
	}

	// Validate review mode.
	switch cfg.ReviewMode {
	case assemble.FullAssembly, "":
		// default
	case assemble.DeltaReview, assemble.SectionUpdate:
		// implemented below
	default:
		fmt.Fprintf(os.Stderr, `{"error": "unknown review mode %q; valid values: full_assembly, delta_review, section_update", "code": 2}`+"\n", cfg.ReviewMode)
		os.Exit(exit.ToolError)
	}

	// Load program context from run state.
	runStatePath := *runStateFlag
	if runStatePath == "" {
		runStatePath = filepath.Join("runs", *programFlag, "latest.json")
	}
	ctx, err := assemble.LoadProgramContext(runStatePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load run state %q: %v — assembling with available context\n", runStatePath, err)
		ctx = &assemble.ProgramContext{Program: *programFlag}
	}

	var doc string
	switch cfg.ReviewMode {
	case assemble.DeltaReview:
		if *previousRunStateFlag == "" {
			fmt.Fprintln(os.Stderr, `{"error": "--previous-run-state is required for delta_review mode", "code": 2}`)
			os.Exit(exit.ToolError)
		}
		prevCtx, err := assemble.LoadProgramContext(*previousRunStateFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not load previous run state %q: %v — treating all clauses as changed\n", *previousRunStateFlag, err)
			prevCtx = &assemble.ProgramContext{}
		}
		doc = assemble.AssembleDelta(cfg, ctx, prevCtx)

	case assemble.SectionUpdate:
		if cfg.SectionTarget == "" {
			fmt.Fprintln(os.Stderr, `{"error": "--section is required for section_update mode (4–10)", "code": 2}`)
			os.Exit(exit.ToolError)
		}
		var secErr error
		doc, secErr = assemble.AssembleSection(cfg, ctx)
		if secErr != nil {
			fmt.Fprintf(os.Stderr, `{"error": %q, "code": 2}`+"\n", secErr.Error())
			os.Exit(exit.ToolError)
		}

	default:
		doc = assemble.Assemble(cfg, ctx)
	}

	if *fmtFlag == "json" {
		// JSON summary mode — emit structured metadata without the full document.
		dataNeededCount := strings.Count(doc, "[DATA NEEDED")
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]interface{}{ //nolint:errcheck
			"standard":          string(cfg.Standard),
			"program":           cfg.Program,
			"clauses_generated": strings.Count(doc, "\n## "),
			"data_needed_count": dataNeededCount,
			"dry_run":           *dryRunFlag,
			"output":            coalesceStr(*outputFlag, "stdout"),
		})
		return
	}

	if *dryRunFlag || *outputFlag == "" {
		fmt.Print(doc)
	} else {
		if err := os.WriteFile(*outputFlag, []byte(doc), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
			os.Exit(exit.ToolError)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", *outputFlag)
	}

	_ = provenance.Write("logs/provenance.jsonl", provenance.Entry{
		Spec:        "functions/management-system-assembler-spec.md",
		Output:      coalesceStr(*outputFlag, "stdout"),
		OutputType:  "other",
		Program:     *programFlag,
		Purpose:     fmt.Sprintf("compound: %s document assembly for %s (%s mode)", *standardFlag, *programFlag, *reviewModeFlag),
		Reusability: provenance.Template,
		QualityGate: provenance.Pass,
		Tool:        "compound",
		ToolVersion: version,
	})
}

func coalesceStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func usage() {
	fmt.Fprintln(os.Stderr, `compound — management system document assembly

Usage:
  compound --program <slug> --standard <standard> [flags]

Flags:
  --program string                Program slug (required)
  --standard string               Standard: iso27001, iso42001, iec62443
  --run-state string              Path to run state JSON (default: runs/[program]/latest.json)
  --previous-run-state string     Path to previous run state JSON (required for delta_review)
  --output string                 Output markdown path (default: stdout)
  --scope string                  Scope statement for the management system
  --review-mode string            full_assembly (default), delta_review, section_update
  --org string                    Organization name for branded output
  --section string                Target clause for section_update mode (4–10, e.g. '8')
  --dry-run                       Print without writing
  --version                       Print version and exit

Examples:
  compound --program iso42001 --standard iso42001 > aims-draft.md
  compound --program iso42001 --standard iso42001 --output docs/AIMS.md
  compound --program iso27001 --standard iso27001 --review-mode delta_review \
           --previous-run-state runs/iso27001/2026-Q2.json
  compound --program iso27001 --standard iso27001 --review-mode section_update --section 8`)
}
