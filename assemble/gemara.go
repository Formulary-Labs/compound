package assemble

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Formulary-Labs/substrate/artifact"
)

// GemaraSet holds loaded gemara artifacts for security plan assembly.
type GemaraSet struct {
	Policies       []PolicyRef
	Catalog        *artifact.ControlCatalog
	CatalogPath    string
	AuditLogs      []AuditLogRef
	RiskCatalogs   []RiskCatalogRef // citation-only; never inlined into plan body
	ClauseMap      map[string]string // policy metadata.id → clause (e.g. "5.2")
	ArtifactHashes map[string]string // path → content fingerprint for delta
}

// PolicyRef pairs a loaded Policy with its source path.
type PolicyRef struct {
	Path   string
	Policy *artifact.Policy
	Clause string // Annex SL clause slot, e.g. "4", "5.2", "5.3", "7.4"
}

// AuditLogRef pairs a loaded AuditLog with its source path.
type AuditLogRef struct {
	Path string
	Log  *artifact.AuditLog
}

// RiskCatalogRef pairs a loaded RiskCatalog with its source path (citation only).
type RiskCatalogRef struct {
	Path    string
	Catalog *artifact.RiskCatalog
}

var clauseIDPattern = regexp.MustCompile(`(?i)(?:clause[-_])?(\d+(?:\.\d+)?)`)

// LoadGemaraSet loads all gemara YAML/JSON files from paths (files or directories).
// Optional clauseMapPath is a YAML/JSON file: { "mappings": { "policy-id": "5.2" } }.
func LoadGemaraSet(paths []string, clauseMapPath string) (*GemaraSet, error) {
	set := &GemaraSet{
		ClauseMap:      map[string]string{},
		ArtifactHashes: map[string]string{},
	}
	if clauseMapPath != "" {
		m, err := loadClauseMap(clauseMapPath)
		if err != nil {
			return nil, err
		}
		set.ClauseMap = m
	}

	files, err := expandArtifactPaths(paths)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no gemara artifact files found in %v", paths)
	}

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", path, err)
		}
		set.ArtifactHashes[path] = fingerprint(data)

		t, err := artifact.DetectType(path)
		if err != nil {
			return nil, fmt.Errorf("detecting type in %q: %w", path, err)
		}

		switch t {
		case artifact.PolicyArtifact:
			p, err := artifact.LoadPolicy(path)
			if err != nil {
				return nil, err
			}
			clause := resolveClause(p.Metadata.Id, p.Metadata.ApplicabilityGroups, set.ClauseMap)
			set.Policies = append(set.Policies, PolicyRef{Path: path, Policy: p, Clause: clause})

		case artifact.ControlCatalogArtifact:
			cat, err := artifact.LoadControlCatalog(path)
			if err != nil {
				return nil, err
			}
			set.Catalog = cat
			set.CatalogPath = path

		case artifact.AuditLogArtifact:
			log, err := artifact.LoadAuditLog(path)
			if err != nil {
				return nil, err
			}
			set.AuditLogs = append(set.AuditLogs, AuditLogRef{Path: path, Log: log})

		case artifact.RiskCatalogArtifact:
			rc, err := artifact.LoadRiskCatalog(path)
			if err != nil {
				return nil, err
			}
			set.RiskCatalogs = append(set.RiskCatalogs, RiskCatalogRef{Path: path, Catalog: rc})

		default:
			// Other types ignored for plan assembly (packet / narrative tools consume them).
			continue
		}
	}

	if len(set.Policies) == 0 && set.Catalog == nil {
		return nil, fmt.Errorf("assemble-plan requires at least one Policy or ControlCatalog in gemara inputs")
	}
	return set, nil
}

func expandArtifactPaths(paths []string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("stat %q: %w", p, err)
		}
		if info.IsDir() {
			entries, err := os.ReadDir(p)
			if err != nil {
				return nil, fmt.Errorf("reading dir %q: %w", p, err)
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				ext := strings.ToLower(filepath.Ext(name))
				if ext != ".yaml" && ext != ".yml" && ext != ".json" {
					continue
				}
				fp := filepath.Join(p, name)
				if !seen[fp] {
					seen[fp] = true
					out = append(out, fp)
				}
			}
			continue
		}
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out, nil
}

type clauseMapFile struct {
	Mappings map[string]string `json:"mappings" yaml:"mappings"`
}

func loadClauseMap(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading clause map %q: %w", path, err)
	}
	var cm clauseMapFile
	if err := json.Unmarshal(data, &cm); err != nil {
		// Accept simple YAML-like "key: value" lines without a full YAML dependency.
		cm.Mappings = parseSimpleYAMLMap(string(data))
		if len(cm.Mappings) == 0 {
			return nil, fmt.Errorf("parsing clause map %q: expected JSON {\"mappings\":{...}} or simple key: value lines", path)
		}
	}
	if cm.Mappings == nil {
		cm.Mappings = map[string]string{}
	}
	return cm.Mappings, nil
}

func parseSimpleYAMLMap(s string) map[string]string {
	out := map[string]string{}
	inMappings := false
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if trimmed == "mappings:" {
			inMappings = true
			continue
		}
		if !inMappings && !strings.Contains(trimmed, ":") {
			continue
		}
		// Nested under mappings: or top-level key: value
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if inMappings && indent == 0 && !strings.HasPrefix(trimmed, "mappings") {
			inMappings = false
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(strings.Trim(parts[0], `"'`))
		v := strings.TrimSpace(strings.Trim(parts[1], `"'`))
		if k == "mappings" || k == "" || v == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func resolveClause(metaID string, groups []artifact.Group, clauseMap map[string]string) string {
	if clauseMap != nil {
		if c, ok := clauseMap[metaID]; ok {
			return normalizeClause(c)
		}
	}
	for _, g := range groups {
		if c := extractClause(g.Id); c != "" {
			return c
		}
		if c := extractClause(g.Title); c != "" {
			return c
		}
	}
	if c := extractClause(metaID); c != "" {
		return c
	}
	return ""
}

func extractClause(s string) string {
	m := clauseIDPattern.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return normalizeClause(m[1])
}

func normalizeClause(c string) string {
	c = strings.TrimSpace(c)
	c = strings.TrimPrefix(strings.ToLower(c), "clause-")
	c = strings.TrimPrefix(c, "clause_")
	c = strings.TrimPrefix(c, "clause ")
	return c
}

func fingerprint(data []byte) string {
	// Stable, non-cryptographic content id for delta comparison.
	var h uint64 = 14695981039346656037
	for _, b := range data {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return fmt.Sprintf("%016x", h)
}

// SetFingerprint returns a stable fingerprint of the entire gemara set for delta mode.
func (s *GemaraSet) SetFingerprint() string {
	if s == nil {
		return ""
	}
	keys := make([]string, 0, len(s.ArtifactHashes))
	for k := range s.ArtifactHashes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+s.ArtifactHashes[k])
	}
	return fingerprint([]byte(strings.Join(parts, "|")))
}

// PoliciesForClause returns policies bound to the given clause (exact or parent).
func (s *GemaraSet) PoliciesForClause(clause string) []PolicyRef {
	if s == nil {
		return nil
	}
	clause = normalizeClause(clause)
	var out []PolicyRef
	for _, p := range s.Policies {
		if p.Clause == clause {
			out = append(out, p)
		}
	}
	// Fallback: unbound single policy applies to 4, 5.2, 5.3.
	if len(out) == 0 && len(s.Policies) == 1 && s.Policies[0].Clause == "" {
		switch clause {
		case "4", "5.2", "5.3":
			return s.Policies
		}
	}
	// Unbound multi-policy: first unbound policy for 5.2 only.
	if len(out) == 0 && clause == "5.2" {
		for _, p := range s.Policies {
			if p.Clause == "" {
				out = append(out, p)
				break
			}
		}
	}
	return out
}
