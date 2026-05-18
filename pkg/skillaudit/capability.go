package skillaudit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type CapabilityAudit struct {
	Verified bool
	Summary  string
}

var markdownRelativeLinkPattern = regexp.MustCompile(`!?\[[^\]]*\]\(([^)]+)\)`)
var actionKeywordPattern = regexp.MustCompile(`(?i)\b(must|should|use|run|return|report|check|avoid|invoke|start|launch|pass|set|send|write|review|verify|search|list|read)\b|使用|运行|返回|报告|检查|避免|不要|必须|调用|启动|通过|设置|发送|写入|审查|验证|搜索|列出|读取|说明|选择|连接|构建|初始化|重置`)

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

	if !hasCapabilityStructure(mainPath) {
		return CapabilityAudit{Verified: false, Summary: "missing capability structure"}
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

func hasCapabilityStructure(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	lines := strings.Split(string(content), "\n")
	inFrontMatter := false
	frontmatterClosed := false
	inCodeFence := false
	activeCodeFence := ""

	headings := 0
	bullets := 0
	numbered := 0
	narrativeLines := 0
	substantiveChars := 0
	hasActionLanguage := false

	for i, raw := range lines {
		line := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if i == 0 && line == "---" {
			inFrontMatter = true
			continue
		}
		if inFrontMatter {
			if line == "---" {
				inFrontMatter = false
				frontmatterClosed = true
			}
			continue
		}
		if !frontmatterClosed {
			continue
		}
		if line == "" {
			continue
		}

		if !inCodeFence {
			if fence := codeFenceDelimiter(line); fence != "" {
				inCodeFence = true
				activeCodeFence = fence
				continue
			}
		} else if strings.HasPrefix(line, activeCodeFence) {
			inCodeFence = false
			activeCodeFence = ""
			continue
		}
		if inCodeFence {
			continue
		}

		if strings.HasPrefix(line, "#") {
			headings++
		}
		if isBulletLine(line) {
			bullets++
		}
		if isNumberedLine(line) {
			numbered++
		}

		normalized := normalizeNarrativeLine(line)
		if normalized == "" {
			continue
		}
		narrativeLines++
		substantiveChars += countMeaningfulRunes(normalized)
		if actionKeywordPattern.MatchString(normalized) {
			hasActionLanguage = true
		}
	}

	if headings+bullets+numbered > 0 && narrativeLines >= 2 {
		return true
	}

	return narrativeLines >= 2 && substantiveChars >= 20 && hasActionLanguage
}

func isBulletLine(line string) bool {
	return strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ")
}

func isNumberedLine(line string) bool {
	if len(line) < 3 || !unicode.IsDigit(rune(line[0])) {
		return false
	}
	idx := 0
	for idx < len(line) && unicode.IsDigit(rune(line[idx])) {
		idx++
	}
	if idx >= len(line) {
		return false
	}
	if line[idx] != '.' && line[idx] != ')' {
		return false
	}
	return idx+1 < len(line) && line[idx+1] == ' '
}

func countMeaningfulRunes(value string) int {
	count := 0
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Han, r) {
			count++
		}
	}
	return count
}
