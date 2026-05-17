package skillaudit

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const maxAuditLines = 160

var structuralTagPattern = regexp.MustCompile(`^</?[A-Z0-9_-]+>$`)

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
	activeCodeFence := ""
	lineCount := 0

	for scanner.Scan() && lineCount < maxAuditLines {
		lineCount++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "---" && activeCodeFence == "" {
			inFrontMatter = !inFrontMatter
			continue
		}
		if inFrontMatter {
			continue
		}
		if activeCodeFence == "" {
			if fence := codeFenceDelimiter(line); fence != "" {
				activeCodeFence = fence
				continue
			}
		} else if strings.HasPrefix(line, activeCodeFence) {
			activeCodeFence = ""
			continue
		}
		if activeCodeFence != "" {
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
	if structuralTagPattern.MatchString(trimmed) {
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

func codeFenceDelimiter(line string) string {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return ""
	}
	first := rune(trimmed[0])
	if first != '`' && first != '~' {
		return ""
	}
	count := 0
	for _, r := range trimmed {
		if r == first {
			count++
			continue
		}
		break
	}
	if count < 3 {
		return ""
	}
	return strings.Repeat(string(first), count)
}

func containsHan(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
