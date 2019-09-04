package formats

import "strings"

func ListLanguages(rows [][]string) map[string]bool {
	headIndex := firstNonempty(rows)
	if headIndex == -1 {
		return nil
	}
	langs := make(map[string]bool)
	for _, cell := range rows[headIndex] {
		l := strings.LastIndexByte(cell, '(')
		r := strings.LastIndexByte(cell, ')')
		if l == -1 || r == -1 || l > r {
			continue
		}
		lang := strings.TrimSpace(cell[l+1 : r])
		if lang == "" || lang == "en" {
			continue
		}
		langs[lang] = true
	}
	return langs
}

func Translation(rows [][]string, lang string) map[string]string {
	headIndex := firstNonempty(rows)
	if headIndex == -1 {
		return nil
	}
	head := rows[headIndex]
	translation := make(map[string]string)
	for en, name := range head {
		if strings.HasSuffix(name, "(en)") || !strings.Contains(name, "::") {
			tr := translationIndex(head, name, lang)
			if tr == -1 {
				continue
			}
			for j := headIndex + 1; j < len(rows); j++ {
				row := rows[j]
				translation[row[en]] = row[tr]
			}
		}
	}
	return translation
}

func translationIndex(head []string, name, lang string) int {
	var prefix, suffix string
	i := strings.Index(name, "::")
	if i == -1 {
		prefix = name + "::"
	} else {
		prefix = name[0 : i+2]
	}
	suffix = "(" + lang + ")"
	for i, cell := range head {
		if strings.HasPrefix(cell, prefix) && strings.HasSuffix(cell, suffix) {
			return i
		}
	}
	return -1
}
