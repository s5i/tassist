package hotkey

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantOK   bool
		wantIdx  int
		wantText string
		wantAS   bool
	}{
		{
			name:     "simple autosend",
			line:     `Hotkey = (1,"vrenx quoll habbo\n")`,
			wantOK:   true,
			wantIdx:  1,
			wantText: "vrenx quoll habbo",
			wantAS:   true,
		},
		{
			name:     "no autosend",
			line:     `Hotkey = (18,"o/")`,
			wantOK:   true,
			wantIdx:  18,
			wantText: "o/",
			wantAS:   false,
		},
		{
			name:     "escaped backslash no autosend",
			line:     `Hotkey = (17,"\\o")`,
			wantOK:   true,
			wantIdx:  17,
			wantText: `\o`,
			wantAS:   false,
		},
		{
			name:     "escaped quote with autosend",
			line:     `Hotkey = (34,"zorpan tio \"Blimmy Woff\n")`,
			wantOK:   true,
			wantIdx:  34,
			wantText: `zorpan tio "Blimmy Woff`,
			wantAS:   true,
		},
		{
			name:     "index zero",
			line:     `Hotkey = (0,"xarbo plim zaffa vorq tukkel gnemm\n")`,
			wantOK:   true,
			wantIdx:  0,
			wantText: "xarbo plim zaffa vorq tukkel gnemm",
			wantAS:   true,
		},
		{
			name:   "non-hotkey line",
			line:   `WindowedMode = (970,473,620,493,no)`,
			wantOK: false,
		},
		{
			name:   "comment line",
			line:   `#Tibiantis configuration file`,
			wantOK: false,
		},
		{
			name:   "empty line",
			line:   ``,
			wantOK: false,
		},
		{
			name:   "malformed no parens",
			line:   `Hotkey = 1,"test"`,
			wantOK: false,
		},
		{
			name:   "malformed no quote",
			line:   `Hotkey = (1,test)`,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, hk, ok := parseLine(tt.line)
			if ok != tt.wantOK {
				t.Fatalf("parseLine(%q): ok = %v, want %v", tt.line, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if idx != tt.wantIdx {
				t.Errorf("idx = %d, want %d", idx, tt.wantIdx)
			}
			if hk.Text != tt.wantText {
				t.Errorf("text = %q, want %q", hk.Text, tt.wantText)
			}
			if hk.Autosend != tt.wantAS {
				t.Errorf("autosend = %v, want %v", hk.Autosend, tt.wantAS)
			}
		})
	}
}

func TestReadConfig(t *testing.T) {
	hk, err := ReadConfig(filepath.Join("testdata", "Tibiantis.cfg"))
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	wantCount := 28
	if len(hk) != wantCount {
		t.Fatalf("len(hotkeys) = %d, want %d", len(hk), wantCount)
	}

	check := func(idx int, wantText string, wantAS bool) {
		t.Helper()
		got, ok := hk[idx]
		if !ok {
			t.Errorf("hotkey[%d]: missing", idx)
			return
		}
		if got.Text != wantText {
			t.Errorf("hotkey[%d].Text = %q, want %q", idx, got.Text, wantText)
		}
		if got.Autosend != wantAS {
			t.Errorf("hotkey[%d].Autosend = %v, want %v", idx, got.Autosend, wantAS)
		}
	}

	check(0, "xarbo plim zaffa vorq tukkel gnemm", true)
	check(9, "yibben", true)
	check(17, `\o`, false)
	check(18, "o/", false)
	check(34, `zorpan tio "Blimmy Woff`, true)
	check(35, `zorpan tio "Woff Blinkenschmidt`, true)

	for _, idx := range []int{19, 20, 21, 22, 23, 30, 32, 33} {
		if _, ok := hk[idx]; ok {
			t.Errorf("hotkey[%d]: should not exist", idx)
		}
	}
}

func TestWriteConfig(t *testing.T) {
	src := filepath.Join("testdata", "Tibiantis.cfg")
	orig, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "Tibiantis.cfg")
	if err := os.WriteFile(cfgPath, orig, 0644); err != nil {
		t.Fatalf("write tmp cfg: %v", err)
	}

	newHotkeys := map[int]Hotkey{
		0:  {Text: "replaced zero", Autosend: true},
		5:  {Text: "replaced five", Autosend: false},
		34: {Text: `spell "Target Name`, Autosend: true},
	}

	if err := WriteConfig(cfgPath, newHotkeys); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	bakPath := cfgPath + ".bak"
	bak, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("backup not created: %v", err)
	}
	if string(bak) != string(orig) {
		t.Error("backup content does not match original")
	}

	written, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read written cfg: %v", err)
	}
	content := string(written)

	if strings.Contains(content, "vrenx quoll") {
		t.Error("old hotkey text still present in written config")
	}

	if !strings.Contains(content, `Hotkey = (0,"replaced zero\n")`) {
		t.Error("hotkey 0 not written correctly")
	}
	if !strings.Contains(content, `Hotkey = (5,"replaced five")`) {
		t.Error("hotkey 5 (no autosend) not written correctly")
	}
	if !strings.Contains(content, `Hotkey = (34,"spell \"Target Name\n")`) {
		t.Error("hotkey 34 (escaped quote) not written correctly")
	}

	lines := strings.Split(content, "\n")
	var hotkeyStart, windowedIdx int
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Hotkey") && hotkeyStart == 0 {
			hotkeyStart = i
		}
		if strings.HasPrefix(trimmed, "WindowedMode") {
			windowedIdx = i
			break
		}
	}
	if windowedIdx > 0 && hotkeyStart >= windowedIdx {
		t.Error("hotkey lines should appear before WindowedMode")
	}

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "Buddy =") {
			t.Error("Buddy entries should not appear (testdata has none)")
			break
		}
	}
}

func TestWriteConfigReadBack(t *testing.T) {
	src := filepath.Join("testdata", "Tibiantis.cfg")
	orig, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "Tibiantis.cfg")
	if err := os.WriteFile(cfgPath, orig, 0644); err != nil {
		t.Fatalf("write tmp cfg: %v", err)
	}

	hotkeys := map[int]Hotkey{
		0:  {Text: "simple text", Autosend: true},
		3:  {Text: `with "quotes"`, Autosend: true},
		17: {Text: `\o`, Autosend: false},
		18: {Text: "o/", Autosend: false},
		35: {Text: `cast "Some Name`, Autosend: true},
	}

	if err := WriteConfig(cfgPath, hotkeys); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	readBack, err := ReadConfig(cfgPath)
	if err != nil {
		t.Fatalf("ReadConfig after write: %v", err)
	}

	if len(readBack) != len(hotkeys) {
		t.Fatalf("readback count = %d, want %d", len(readBack), len(hotkeys))
	}

	for idx, want := range hotkeys {
		got, ok := readBack[idx]
		if !ok {
			t.Errorf("hotkey[%d]: missing after readback", idx)
			continue
		}
		if got.Text != want.Text {
			t.Errorf("hotkey[%d].Text = %q, want %q", idx, got.Text, want.Text)
		}
		if got.Autosend != want.Autosend {
			t.Errorf("hotkey[%d].Autosend = %v, want %v", idx, got.Autosend, want.Autosend)
		}
	}
}

func TestEscapeUnescapeRoundtrip(t *testing.T) {
	inputs := []string{
		"simple text",
		`text with "quotes"`,
		`backslash \o`,
		"tabs\there",
		"newlines\nhere",
		`mixed \"quotes\" and \\slashes`,
	}

	for _, input := range inputs {
		escaped := escapeCfgString(input)
		quoted := `"` + escaped + `"`
		got := unescapeCfgString(quoted)
		if got != input {
			t.Errorf("roundtrip(%q): escaped=%q, unescaped=%q", input, escaped, got)
		}
	}
}

func TestFormatLineParseLine(t *testing.T) {
	tests := []struct {
		idx int
		hk  Hotkey
	}{
		{0, Hotkey{Text: "hello world", Autosend: true}},
		{17, Hotkey{Text: `\o`, Autosend: false}},
		{34, Hotkey{Text: `spell "Player Name`, Autosend: true}},
		{11, Hotkey{Text: "plain", Autosend: false}},
	}

	for _, tt := range tests {
		line := formatLine(tt.idx, tt.hk)
		gotIdx, gotHk, ok := parseLine(line)
		if !ok {
			t.Errorf("formatLine(%d, %+v) = %q: parseLine failed", tt.idx, tt.hk, line)
			continue
		}
		if gotIdx != tt.idx {
			t.Errorf("idx = %d, want %d (line: %q)", gotIdx, tt.idx, line)
		}
		if gotHk.Text != tt.hk.Text {
			t.Errorf("text = %q, want %q (line: %q)", gotHk.Text, tt.hk.Text, line)
		}
		if gotHk.Autosend != tt.hk.Autosend {
			t.Errorf("autosend = %v, want %v (line: %q)", gotHk.Autosend, tt.hk.Autosend, line)
		}
	}
}
