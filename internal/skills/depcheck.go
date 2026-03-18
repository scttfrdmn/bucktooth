package skills

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DepResult holds the dependency-check outcome for a single skill.
type DepResult struct {
	SkillName   string   `json:"skill_name"`
	MissingBins []string `json:"missing_bins,omitempty"`
	MissingEnv  []string `json:"missing_env,omitempty"`
	OK          bool     `json:"ok"`
}

// SkillMeta holds the parsed YAML frontmatter from a SKILL.md file.
type SkillMeta struct {
	Name string   `yaml:"name"`
	Bins []string `yaml:"bins"`
	Env  []string `yaml:"env"`
}

// DepChecker validates SKILL.md dependency declarations for a list of search paths.
type DepChecker struct {
	searchPaths []string
}

// NewDepChecker creates a DepChecker that scans the given directories for SKILL.md files.
func NewDepChecker(searchPaths []string) *DepChecker {
	return &DepChecker{searchPaths: searchPaths}
}

// ParseSkillMeta parses the YAML frontmatter from a SKILL.md file's content.
// Returns (nil, nil) when no frontmatter delimiters are found (no requirements).
func ParseSkillMeta(content string) (*SkillMeta, error) {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, nil
	}
	var meta SkillMeta
	if err := yaml.Unmarshal([]byte(parts[1]), &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// Check returns a DepResult for the named skill.
// If no SKILL.md is found in any search path, the skill is considered to have no requirements (OK: true).
func (d *DepChecker) Check(skillName string) DepResult {
	result := DepResult{SkillName: skillName, OK: true}

	// Locate SKILL.md for this skill across all search paths.
	var meta *SkillMeta
	for _, searchPath := range d.searchPaths {
		skillMDPath := filepath.Join(searchPath, skillName, "SKILL.md")
		data, err := os.ReadFile(skillMDPath)
		if err != nil {
			continue
		}
		parsed, err := ParseSkillMeta(string(data))
		if err != nil {
			continue
		}
		meta = parsed
		break
	}

	if meta == nil {
		// No SKILL.md found — no declared requirements.
		return result
	}

	// Verify each required binary is on PATH.
	for _, bin := range meta.Bins {
		if _, err := exec.LookPath(bin); err != nil {
			result.MissingBins = append(result.MissingBins, bin)
		}
	}

	// Verify each required environment variable is set.
	for _, env := range meta.Env {
		if os.Getenv(env) == "" {
			result.MissingEnv = append(result.MissingEnv, env)
		}
	}

	result.OK = len(result.MissingBins) == 0 && len(result.MissingEnv) == 0
	return result
}

// CheckAll returns a DepResult for each skill name in the provided slice.
// Results are returned in the same order as the input slice.
func (d *DepChecker) CheckAll(skillNames []string) []DepResult {
	results := make([]DepResult, len(skillNames))
	for i, name := range skillNames {
		results[i] = d.Check(name)
	}
	return results
}
