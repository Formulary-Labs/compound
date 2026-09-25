// compound assembles Annex SL-structured management system documents
// (ISMS, AIMS, CSMS) from program artifacts or gemara layers.
//
// Usage:
//
//	compound --program <slug> --standard <iso27001|iso42001|iec62443> [flags]
//	compound --assemble-plan --program <slug> --gemara <dir|file>... [flags]
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

const version = "0.2.0"

// multiFlag collects repeated flag values.
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func main() {
	var (
		programFlag          = flag.String("program", "", "Program slug (required)")
		standardFlag         = flag.String("standard", "", "Standard: iso27001, iso42001, iec62443")
		runStateFlag         = flag.String("run-state", "", "Path to run state JSON (default: runs/[program]/latest.json)")
		previousRunStateFlag = flag.String("previous-run-state", "", "Path to previous run state JSON (required for delta_review mode)")
		outputFlag           = flag.String("output", "", "Output markdown path (default: stdout)")
		reportFlag           = flag.String("report", "", "Assembly report JSON path (assemble-plan only)")
		fmtFlag              = flag.String("format", "md", "Output format: md (default), json (JSON summary only)")
		scopeFlag            = flag.String("scope", "", "Scope statement for the management system")
		reviewModeFlag       = flag.String("review-mode", "full_assembly", "Review mode: full_assembly, delta_review, section_update")
		orgNameFlag          = flag.String("org", "", "Organization name for branded output")
		sectionFlag          = flag.String("section", "", "Target clause for section_update mode (4–10, e.g. '8')")
		assemblePlanFlag     = flag.Bool("assemble-plan", false, "Assemble security plan from gemara artifacts")
		clauseMapFlag        = flag.String("clause-map", "", "Optional policy-id → clause map (JSON or simple YAML)")
		priorGemaraFlag      = flag.String("prior-gemara", "", "Prior gemara dir/files for assemble-plan delta_review")
		dryRunFlag           = flag.Bool("dry-run", false, "Print what would be written without writing")
		versionFlag          = flag.Bool("version", false, "Print version and exit")
		gemaraFlags          multiFlag
	)
	flag.Var(&gemaraFlags, "gemara", "Gemara artifact file or directory (repeatable; required for --assemble-plan)")
	flag.Usage = usage
	flag.Parse()

	// Positional args also accepted as gemara paths when --assemble-plan is set.
	if *assemblePlanFlag {
		gemaraFlags = append(gemaraFlags, flag.Args()...)
	}

	if *versionFlag {
		fmt.Printf("compound version %s\n", version)
		os.Exit(exit.OK)
	}

	if *programFlag == "" {
		fmt.Fprintln(os.Stderr, `{"error": "--program is required", "code": 2}`)
		flag.Usage()
		os.Exit(exit.ToolError)
	}

	if *assemblePlanFlag {
		runAssemblePlan(*programFlag, *standardFlag, gemaraFlags, *clauseMapFlag, *priorGemaraFlag,
			*reviewModeFlag, *scopeFlag, *orgNameFlag, *outputFlag, *reportFlag, *fmtFlag, *dryRunFlag)
		return
	}

	if *standardFlag == "" {
		fmt.Fprintln(os.Stderr, `{"error": "--standard is required (or use --assemble-plan with gemara inputs)", "code": 2}`)
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

	switch cfg.ReviewMode {
	case assemble.FullAssembly, "":
	case assemble.DeltaReview, assemble.SectionUpdate:
	default:
		fmt.Fprintf(os.Stderr, `{"error": "unknown review mode %q; valid values: full_assembly, delta_review, section_update", "code": 2}`+"\n", cfg.ReviewMode)
		os.Exit(exit.ToolError)
	}

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

	emitOutput(doc, nil, cfg, *fmtFlag, *outputFlag, "", *dryRunFlag, *programFlag, *standardFlag, *reviewModeFlag)
}

func runAssemblePlan(program, standard string, gemaraPaths []string, clauseMap, priorGemara, reviewMode, scope, org, output, reportPath, format string, dryRun bool) {
	if len(gemaraPaths) == 0 {
		fmt.Fprintln(os.Stderr, `{"error": "--gemara is required for --assemble-plan (file or directory)", "code": 2}`)
		os.Exit(exit.ToolError)
	}

	set, err := assemble.LoadGemaraSet(gemaraPaths, clauseMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"error": %q, "code": 2}`+"\n", err.Error())
		os.Exit(exit.ToolError)
	}

	cfg := assemble.DocumentConfig{
		Program:       program,
		Standard:      assemble.Standard(standard),
		Applicability: scope,
		ReviewMode:    assemble.ReviewMode(reviewMode),
	}
	if org != "" {
		cfg.BrandedOutput = true
		cfg.OrgName = org
	}
	if cfg.Standard == "" {
		cfg.Standard = assemble.Standard(standard)
	}

	var result assemble.PlanResult
	switch cfg.ReviewMode {
	case assemble.DeltaReview:
		if priorGemara == "" {
			fmt.Fprintln(os.Stderr, `{"error": "--prior-gemara is required for assemble-plan delta_review", "code": 2}`)
			os.Exit(exit.ToolError)
		}
		priorPaths := strings.Split(priorGemara, ",")
		prev, err := assemble.LoadGemaraSet(priorPaths, clauseMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, `{"error": %q, "code": 2}`+"\n", err.Error())
			os.Exit(exit.ToolError)
		}
		result = assemble.AssemblePlanDelta(cfg, set, prev)
	default:
		result = assemble.AssemblePlan(cfg, set)
	}

	emitOutput(result.Markdown, &result.Report, cfg, format, output, reportPath, dryRun, program, string(cfg.Standard), "assemble-plan")
}

func emitOutput(doc string, report *assemble.AssemblyReport, cfg assemble.DocumentConfig, format, output, reportPath string, dryRun bool, program, standard, mode string) {
	if format == "json" {
		payload := map[string]interface{}{
			"standard":          string(cfg.Standard),
			"program":           cfg.Program,
			"clauses_generated": strings.Count(doc, "\n## "),
			"data_needed_count": strings.Count(doc, "[DATA NEEDED"),
			"dry_run":           dryRun,
			"output":            coalesceStr(output, "stdout"),
			"mode":              mode,
		}
		if report != nil {
			payload["assembly_report"] = report
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(payload)
		return
	}

	if !dryRun && report != nil && reportPath != "" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error marshaling assembly report: %v\n", err)
			os.Exit(exit.ToolError)
		}
		if err := os.WriteFile(reportPath, data, 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "error writing assembly report: %v\n", err)
			os.Exit(exit.ToolError)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", reportPath)
	}

	if dryRun || output == "" {
		fmt.Print(doc)
	} else {
		if err := os.WriteFile(output, []byte(doc), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
			os.Exit(exit.ToolError)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", output)
	}

	_ = provenance.Write("logs/provenance.jsonl", provenance.Entry{
		Spec:        "functions/management-system-assembler-spec.md",
		Output:      coalesceStr(output, "stdout"),
		OutputType:  "other",
		Program:     program,
		Purpose:     fmt.Sprintf("compound: %s document assembly for %s (%s)", standard, program, mode),
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
  compound --assemble-plan --program <slug> --gemara <dir|file> [--standard ...] [flags]

Flags:
  --program string                Program slug (required)
  --standard string               Standard: iso27001, iso42001, iec62443
  --assemble-plan                 Assemble security plan from gemara artifacts
  --gemara string                 Gemara file or directory (repeatable; assemble-plan)
  --clause-map string             Policy id → clause map for assemble-plan
  --prior-gemara string           Prior gemara paths (comma-separated) for delta_review
  --report string                 Write assembly-report.json (assemble-plan)
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
  compound --assemble-plan --program iso27001 --standard iso27001 \
           --gemara data/iso27001/gemara/ --output docs/ISMS-security-plan.md \
           --report docs/assembly-report.json
  compound --program iso27001 --standard iso27001 --review-mode delta_review \
           --previous-run-state runs/iso27001/2026-Q2.json`)
}
