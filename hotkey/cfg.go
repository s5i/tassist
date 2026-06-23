package hotkey

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Hotkey struct {
	Text     string `yaml:"text" json:"text"`
	Autosend bool   `yaml:"autosend" json:"autosend"`
}

func ReadConfig(path string) (map[int]Hotkey, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open cfg: %w", err)
	}
	defer f.Close()

	hotkeys := make(map[int]Hotkey)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		idx, hk, ok := parseLine(sc.Text())
		if !ok {
			continue
		}
		hotkeys[idx] = hk
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read cfg: %w", err)
	}
	return hotkeys, nil
}

func parseLine(line string) (int, Hotkey, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "Hotkey") {
		return 0, Hotkey{}, false
	}

	eqIdx := strings.Index(line, "=")
	if eqIdx < 0 {
		return 0, Hotkey{}, false
	}
	rest := strings.TrimSpace(line[eqIdx+1:])

	if len(rest) < 2 || rest[0] != '(' || rest[len(rest)-1] != ')' {
		return 0, Hotkey{}, false
	}
	rest = rest[1 : len(rest)-1]

	commaIdx := strings.Index(rest, ",")
	if commaIdx < 0 {
		return 0, Hotkey{}, false
	}

	idx, err := strconv.Atoi(strings.TrimSpace(rest[:commaIdx]))
	if err != nil {
		return 0, Hotkey{}, false
	}

	quoted := strings.TrimSpace(rest[commaIdx+1:])
	if len(quoted) < 2 || quoted[0] != '"' {
		return 0, Hotkey{}, false
	}

	text := unescapeCfgString(quoted)

	autosend := strings.HasSuffix(text, "\n")
	text = strings.TrimRight(text, "\n")

	return idx, Hotkey{Text: text, Autosend: autosend}, true
}

func unescapeCfgString(quoted string) string {
	var b strings.Builder
	i := 1
	for i < len(quoted) {
		if quoted[i] == '\\' && i+1 < len(quoted) {
			next := quoted[i+1]
			switch next {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte('\\')
				b.WriteByte(next)
			}
			i += 2
			continue
		}
		if quoted[i] == '"' {
			break
		}
		b.WriteByte(quoted[i])
		i++
	}
	return b.String()
}

func WriteConfig(path string, hotkeys map[int]Hotkey) error {
	orig, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read cfg for backup: %w", err)
	}
	if err := os.WriteFile(path+".bak", orig, 0644); err != nil {
		return fmt.Errorf("write cfg backup: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open cfg for read: %w", err)
	}

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Hotkey") && strings.Contains(trimmed, "=") {
			continue
		}
		lines = append(lines, line)
	}
	if err := sc.Err(); err != nil {
		f.Close()
		return fmt.Errorf("read cfg: %w", err)
	}
	f.Close()

	var insertIdx int
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "WindowedMode") {
			insertIdx = i
			break
		}
	}
	if insertIdx == 0 {
		insertIdx = len(lines)
	}

	var hotkeyLines []string
	for idx := 0; idx <= 35; idx++ {
		hk, ok := hotkeys[idx]
		if !ok {
			continue
		}
		hotkeyLines = append(hotkeyLines, formatLine(idx, hk))
	}

	result := make([]string, 0, len(lines)+len(hotkeyLines))
	result = append(result, lines[:insertIdx]...)
	result = append(result, hotkeyLines...)
	result = append(result, lines[insertIdx:]...)

	return os.WriteFile(path, []byte(strings.Join(result, "\n")+"\n"), 0644)
}

func formatLine(idx int, hk Hotkey) string {
	escaped := escapeCfgString(hk.Text)
	if hk.Autosend {
		escaped += `\n`
	}
	return fmt.Sprintf(`Hotkey = (%d,"%s")`, idx, escaped)
}

func escapeCfgString(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
