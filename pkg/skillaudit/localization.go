package skillaudit

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const maxAuditLines = 160

// LocalizationChecked returns true only when the copied skill content
// reads like a fully Chinese-localized skill document instead of a mixed or
// partially translated copy.
func LocalizationChecked(root string, id string) bool {
	markdownPath := filepath.Join(root, id+".md")
	if filePassesChineseAudit(markdownPath) {
		return true
	}
	skillPath := filepath.Join(root, id, "SKILL.md")
	if filePassesChineseAudit(skillPath) {
		return true
	}
	return false
}

func filePassesChineseAudit(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	meaningfulChineseLines := 0
	meaningfulEnglishLines := 0
	inFrontMatter := false
	inCodeFence := false
	lineCount := 0

	for scanner.Scan() && lineCount < maxAuditLines {
		lineCount++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "---" && !inCodeFence {
			inFrontMatter = !inFrontMatter
			continue
		}
		if inFrontMatter {
			continue
		}
		if strings.HasPrefix(line, "```") {
			inCodeFence = !inCodeFence
			continue
		}
		if inCodeFence {
			continue
		}

		normalized := normalizeNarrativeLine(line)
		if normalized == "" {
			continue
		}

		switch {
		case containsHan(normalized):
			meaningfulChineseLines++
		case looksLikeEnglishNarrative(normalized):
			meaningfulEnglishLines++
		}
	}

	return meaningfulChineseLines >= 3 && meaningfulEnglishLines == 0
}

func normalizeNarrativeLine(line string) string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimLeft(trimmed, "#>-*0123456789. \t")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "`") || strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return ""
	}
	if strings.Contains(trimmed, "://") && !containsHan(trimmed) {
		return ""
	}
	return trimmed
}

func looksLikeEnglishNarrative(line string) bool {
	asciiLetters := 0
	for _, r := range line {
		if r <= unicode.MaxASCII && unicode.IsLetter(r) {
			asciiLetters++
		}
	}
	return asciiLetters >= 4
}

func containsHan(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
