package lint

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds project-level vocabulary rules from .gforge.yml.
type Config struct {
	Lint struct {
		AllowTerms []string `yaml:"allow_terms"`
		DenyTerms  []string `yaml:"deny_terms"`
	} `yaml:"lint"`
}

// LoadConfig walks up from startDir until it finds .gforge.yml or reaches
// the filesystem root. Returns an empty Config (no error) if no file found.
func LoadConfig(startDir string) (Config, error) {
	dir := startDir
	for {
		candidate := filepath.Join(dir, ".gforge.yml")
		data, err := os.ReadFile(candidate)
		if err == nil {
			var cfg Config
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return Config{}, err
			}
			return cfg, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return Config{}, nil
}

// buildForbidden constructs the runtime forbidden list for a single lint run.
// It starts from the package-level forbiddenPatterns, appends any deny_terms
// from cfg, then removes any entry that appears in cfg's allow_terms.
// The package-level var is never mutated.
func buildForbidden(cfg Config) []string {
	allowed := make(map[string]bool, len(cfg.Lint.AllowTerms))
	for _, t := range cfg.Lint.AllowTerms {
		allowed[t] = true
	}

	base := append([]string(nil), forbiddenPatterns...)
	base = append(base, cfg.Lint.DenyTerms...)

	result := base[:0:0]
	for _, t := range base {
		if !allowed[t] {
			result = append(result, t)
		}
	}
	return result
}
