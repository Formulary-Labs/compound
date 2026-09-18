// compound assembles Annex SL-structured management system documents
// (ISMS, AIMS, CSMS) from program artifacts.
//
// Usage:
//
//	compound --program <slug> --standard <iso27001|iso42001|iec62443> [flags]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Formulary-Labs/compound/assemble"
	"github.com/Formulary-Labs/substrate/exit"
	"github.com/Formulary-Labs/substrate/provenance"
)

const version = "0.1.0"

func main() {
	var (
		programFlag   = flag.String("program", "", "Program slug (required)")
		standardFlag  = flag.String("standard", "", "Standard: iso27001, iso42001, iec62443 (required)")
		runStateFlag  = flag.String("run-state", "", "Path to run state JSON (default: runs/[program]/latest.json)")
		outputFlag    = flag.String("output", "", "Output markdown path (default: stdout)")
		scopeFlag     = flag.String("scope", "", "Scope statement for the management system")
		reviewModeFlag = flag.String("review-mode", "full_assembly", "Review mode: full_assembly, delta_review, section_update")
		orgNameFlag   = flag.String("org", "", "Organization name for branded output")
		sectionFlag   = flag.String("section", "", "Target section for section_update mode (e.g. '8')")
		dryRunFlag    = flag.Bool("dry-run", false, "Print what would be written without writing")
		versionFlag   = flag.Bool("version", false, "Print version and exit")
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

	doc := assemble.Assemble(cfg, ctx)

	if *dryRunFlag || *outputFlag == "" {
		fmt.Print(doc)
	} else {
		if err := os.WriteFile(*outputFlag, []byte(doc), 0o644); err != nil {
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
  --program string       Program slug (required)
  --standard string      Standard: iso27001, iso42001, iec62443
  --run-state string     Path to run state JSON (default: runs/[program]/latest.json)
  --output string        Output markdown path (default: stdout)
  --scope string         Scope statement for the management system
  --review-mode string   full_assembly, delta_review, section_update (default: full_assembly)
  --org string           Organization name for branded output
  --section string       Target section for section_update mode
  --dry-run              Print without writing
  --version              Print version and exit

Examples:
  compound --program iso42001 --standard iso42001 > aims-draft.md
  compound --program iso42001 --standard iso42001 --output docs/AIMS.md
  compound --program iso27001 --standard iso27001 --scope "Product X SaaS platform"`)
}
