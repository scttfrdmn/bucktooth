package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSkillMeta_WithFrontmatter(t *testing.T) {
	content := `---
name: my-skill
bins: [jq, curl]
env: [MY_API_KEY]
---
Some skill instructions here.
`
	meta, err := ParseSkillMeta(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}
	if meta.Name != "my-skill" {
		t.Errorf("name: got %q, want %q", meta.Name, "my-skill")
	}
	if len(meta.Bins) != 2 {
		t.Errorf("bins: got %v, want [jq curl]", meta.Bins)
	}
	if len(meta.Env) != 1 || meta.Env[0] != "MY_API_KEY" {
		t.Errorf("env: got %v, want [MY_API_KEY]", meta.Env)
	}
}

func TestParseSkillMeta_NoFrontmatter(t *testing.T) {
	meta, err := ParseSkillMeta("Just some markdown without frontmatter.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta != nil {
		t.Errorf("expected nil meta for content without frontmatter, got %+v", meta)
	}
}

func TestDepChecker_AllDepsMet(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Use a binary that's guaranteed to exist on PATH.
	content := "---\nname: test-skill\nbins: [sh]\nenv: []\n---\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	checker := NewDepChecker([]string{dir})
	result := checker.Check("test-skill")

	if !result.OK {
		t.Errorf("expected OK=true, got MissingBins=%v MissingEnv=%v", result.MissingBins, result.MissingEnv)
	}
	if len(result.MissingBins) != 0 {
		t.Errorf("expected no missing bins, got %v", result.MissingBins)
	}
}

func TestDepChecker_MissingBinary(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "bin-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: bin-skill\nbins: [definitely-not-a-real-binary-xyz123]\n---\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	checker := NewDepChecker([]string{dir})
	result := checker.Check("bin-skill")

	if result.OK {
		t.Error("expected OK=false for missing binary")
	}
	if len(result.MissingBins) != 1 || result.MissingBins[0] != "definitely-not-a-real-binary-xyz123" {
		t.Errorf("expected MissingBins=[definitely-not-a-real-binary-xyz123], got %v", result.MissingBins)
	}
}

func TestDepChecker_MissingEnvVar(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "env-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Use an env var name that definitely won't be set in CI.
	envVarName := "BUCKTOOTH_TEST_MISSING_ENV_XYZ999"
	os.Unsetenv(envVarName)

	content := "---\nname: env-skill\nenv: [" + envVarName + "]\n---\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	checker := NewDepChecker([]string{dir})
	result := checker.Check("env-skill")

	if result.OK {
		t.Error("expected OK=false for missing env var")
	}
	if len(result.MissingEnv) != 1 || result.MissingEnv[0] != envVarName {
		t.Errorf("expected MissingEnv=[%s], got %v", envVarName, result.MissingEnv)
	}
}

func TestDepChecker_NoSKILLmd(t *testing.T) {
	dir := t.TempDir()
	checker := NewDepChecker([]string{dir})
	result := checker.Check("nonexistent-skill")

	if !result.OK {
		t.Errorf("expected OK=true when no SKILL.md found, got %+v", result)
	}
	if result.SkillName != "nonexistent-skill" {
		t.Errorf("skill_name: got %q, want %q", result.SkillName, "nonexistent-skill")
	}
}

func TestDepChecker_CheckAll(t *testing.T) {
	dir := t.TempDir()

	// Skill with met deps
	okDir := filepath.Join(dir, "ok-skill")
	os.MkdirAll(okDir, 0755)
	os.WriteFile(filepath.Join(okDir, "SKILL.md"), []byte("---\nname: ok-skill\nbins: [sh]\n---\n"), 0644)

	// Skill with missing dep
	badDir := filepath.Join(dir, "bad-skill")
	os.MkdirAll(badDir, 0755)
	os.WriteFile(filepath.Join(badDir, "SKILL.md"), []byte("---\nname: bad-skill\nbins: [no-such-bin-abc]\n---\n"), 0644)

	checker := NewDepChecker([]string{dir})
	results := checker.CheckAll([]string{"ok-skill", "bad-skill", "missing-skill"})

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if !results[0].OK {
		t.Errorf("ok-skill should be OK, got %+v", results[0])
	}
	if results[1].OK {
		t.Errorf("bad-skill should not be OK, got %+v", results[1])
	}
	if !results[2].OK {
		t.Errorf("missing-skill (no SKILL.md) should be OK, got %+v", results[2])
	}
}
