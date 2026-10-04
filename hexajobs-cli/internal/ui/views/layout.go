// Package views contains pure rendering functions. It performs no I/O.
package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"hexajobs.dev/hexajobs-cli/internal/client"
)

type Styles struct{ Title, Text, Muted, Selected, Green, Yellow, Red, Border lipgloss.Style }

// Fit clips by terminal cells, including wide glyphs and ANSI styles, and fixes
// the line count. Every top-level view is bounded even during rapid resizes.
func Fit(text string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(text, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "")
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func Line(text string, width int) string {
	return ansi.Truncate(client.CleanText(text), max(0, width), "…")
}
func Wrap(text string, width int) string {
	return lipgloss.NewStyle().Width(max(1, width)).Render(client.CleanText(text))
}
func Box(content string, width, height int, s Styles) string {
	if width < 4 || height < 3 {
		return Fit(content, width, height)
	}
	return Fit(s.Border.Width(width-2).Height(height-2).Render(Fit(content, width-2, height-2)), width, height)
}

var translations = map[string]map[string]string{
	"id": {
		"Search": "Cari", "Radar Map": "Peta Radar", "Web3 & Bounty": "Web3 & Bounty", "History": "Riwayat", "Settings": "Pengaturan", "Donate": "Dukungan",
		"FIND YOUR NEXT MOVE": "TEMUKAN PELUANG BERIKUTNYA", "Search jobs": "Cari lowongan", "Keywords, skills, or company": "Kata kunci, skill, atau perusahaan", "Live Market Pulse": "Ringkasan Pasar",
		"Observed sample": "Sampel teramati", "No market data yet": "Belum ada data pasar", "Press Enter to search": "Tekan Enter untuk mencari", "Language": "Bahasa", "Target region": "Target wilayah", "ScamShield": "ScamShield",
		"Results": "Hasil", "No listings to show": "Tidak ada lowongan", "No history yet": "Riwayat masih kosong", "Local history": "Riwayat lokal", "AI Pitch / local draft": "AI Pitch / draf lokal",
		"Apply": "Buka tautan", "Save": "Simpan", "Back": "Kembali", "Inspect": "Detail", "Ready": "Siap", "Loading": "Memuat", "ON": "AKTIF", "OFF": "NONAKTIF",
		"Support open source": "Dukung open source", "Regional radar": "Radar wilayah", "Settings saved": "Pengaturan tersimpan",
	},
	"jp": {
		"Search": "検索", "Radar Map": "地域マップ", "Web3 & Bounty": "Web3・報奨金", "History": "履歴", "Settings": "設定", "Donate": "支援",
		"FIND YOUR NEXT MOVE": "次のチャンスを見つけよう", "Search jobs": "求人検索", "Keywords, skills, or company": "キーワード・スキル・企業名", "Live Market Pulse": "市場の概要",
		"Observed sample": "取得済みサンプル", "No market data yet": "市場データはまだありません", "Press Enter to search": "Enter で検索", "Language": "言語", "Target region": "対象地域", "ScamShield": "ScamShield",
		"Results": "検索結果", "No listings to show": "求人はありません", "No history yet": "履歴はまだありません", "Local history": "ローカル履歴", "AI Pitch / local draft": "AI Pitch / ローカル下書き",
		"Apply": "リンクを開く", "Save": "保存", "Back": "戻る", "Inspect": "詳細", "Ready": "準備完了", "Loading": "読み込み中", "ON": "有効", "OFF": "無効",
		"Support open source": "オープンソースを支援", "Regional radar": "地域レーダー", "Settings saved": "設定を保存しました",
	},
}

func Tr(language, key string) string {
	if value := translations[language][key]; value != "" {
		return value
	}
	return key
}

func Menu(language string) []string {
	return []string{"01. " + Tr(language, "Search"), "02. " + Tr(language, "Radar Map"), "03. " + Tr(language, "Web3 & Bounty"), "04. " + Tr(language, "History"), "05. " + Tr(language, "Settings"), "06. " + Tr(language, "Donate")}
}

func Sidebar(selected int, focused bool, width, height int, language string, s Styles) string {
	lines := []string{s.Muted.Render("NAVIGATION"), ""}
	for i, item := range Menu(language) {
		prefix := "  "
		style := s.Text
		if i == selected {
			prefix = "> "
			if focused {
				style = s.Selected
			} else {
				style = s.Title
			}
		}
		lines = append(lines, style.Render(Line(prefix+item, width-2)), "")
	}
	return Box(strings.Join(lines, "\n"), width, height, s)
}
