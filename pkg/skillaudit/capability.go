package skillaudit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type CapabilityAudit struct {
	Verified bool
	Summary  string
}

var markdownRelativeLinkPattern = regexp.MustCompile(`!?\[[^\]]*\]\(([^)]+)\)`)

func VerifyCapability(root string, id string) CapabilityAudit {
	mainPath, skillDir, err := resolveSkillPrimaryPath(root, id)
	if err != nil {
		return CapabilityAudit{Verified: false, Summary: err.Error()}
	}

	name, description, err := parseSkillFrontmatter(mainPath)
	if err != nil {
		return CapabilityAudit{Verified: false, Summary: err.Error()}
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(description) == "" {
		return CapabilityAudit{Verified: false, Summary: "missing required frontmatter fields"}
	}

	if skillDir != "" {
		if _, err := os.Stat(skillDir); err != nil {
			return CapabilityAudit{Verified: false, Summary: "skill directory is missing"}
		}
	}

	missingRef, err := firstMissingLocalReference(mainPath)
	if err != nil {
		return CapabilityAudit{Verified: false, Summary: err.Error()}
	}
	if missingRef != "" {
		return CapabilityAudit{Verified: false, Summary: fmt.Sprintf("missing referenced file: %s", missingRef)}
	}

	return CapabilityAudit{Verified: true, Summary: "capability verified"}
}

func resolveSkillPrimaryPath(root string, id string) (string, string, error) {
	singleFile := filepath.Join(root, id+".md")
	if info, err := os.Stat(singleFile); err == nil && !info.IsDir() {
		return singleFile, "", nil
	}

	skillDir := filepath.Join(root, id)
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if info, err := os.Stat(skillFile); err == nil && !info.IsDir() {
		return skillFile, skillDir, nil
	}

	return "", "", fmt.Errorf("missing primary skill document")
}

func parseSkillFrontmatter(path string) (string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", fmt.Errorf("open skill file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return "", "", fmt.Errorf("missing frontmatter")
	}
	if strings.TrimSpace(scanner.Text()) != "---" {
		return "", "", fmt.Errorf("missing frontmatter")
	}

	var name string
	var description string
	closed := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" {
			closed = true
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		value = strings.TrimSpace(strings.Trim(value, `"'`))
		switch key {
		case "name":
			name = value
		case "description":
			description = value
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("read skill file: %w", err)
	}
	if !closed {
		return "", "", fmt.Errorf("unterminated frontmatter")
	}
	return name, description, nil
}

func firstMissingLocalReference(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read skill file: %w", err)
	}

	baseDir := filepath.Dir(path)
	matches := markdownRelativeLinkPattern.FindAllStringSubmatch(string(content), -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		ref := strings.TrimSpace(match[1])
		if ref == "" || strings.HasPrefix(ref, "#") {
			continue
		}
		lower := strings.ToLower(ref)
		if strings.Contains(lower, "://") || strings.HasPrefix(lower, "mailto:") {
			continue
		}
		ref = strings.TrimSpace(strings.TrimPrefix(ref, "file://"))
		ref = strings.TrimSpace(strings.TrimPrefix(ref, "/"))
		ref = filepath.FromSlash(ref)
		target := filepath.Join(baseDir, ref)
		if _, err := os.Stat(target); err != nil {
			return filepath.ToSlash(ref), nil
		}
	}
	return "", nil
}
