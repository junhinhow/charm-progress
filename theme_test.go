package progress

import (
	"strings"
	"testing"
)

func TestASCIiThemeUsaSomenteASCII(t *testing.T) {
	theme := ASCIiTheme()
	glyphs := map[string]string{
		"FullChar":        theme.FullChar,
		"EmptyChar":       theme.EmptyChar,
		"PendingSymbol":   theme.PendingSymbol,
		"RunningSymbol":   theme.RunningSymbol,
		"DoneSymbol":      theme.DoneSymbol,
		"FailedSymbol":    theme.FailedSymbol,
		"CancelledSymbol": theme.CancelledSymbol,
		"Ellipsis":        theme.Ellipsis,
	}
	for name, glyph := range glyphs {
		if glyph == "" {
			t.Fatalf("%s vazio no tema ASCII", name)
		}
		for _, r := range glyph {
			if r > 127 {
				t.Fatalf("%s contém rune não-ASCII %q (U+%04X)", name, r, r)
			}
		}
	}
	if theme.FullChar != "#" || theme.EmptyChar != "-" {
		t.Fatalf("barra ASCII = %q/%q, quer #/-", theme.FullChar, theme.EmptyChar)
	}
}

func TestDefaultThemeUsaUnicode(t *testing.T) {
	theme := DefaultTheme()
	if !strings.Contains(theme.FullChar, "█") {
		t.Fatalf("FullChar = %q, esperava █", theme.FullChar)
	}
	if !strings.Contains(theme.EmptyChar, "░") {
		t.Fatalf("EmptyChar = %q, esperava ░", theme.EmptyChar)
	}
	if theme.Ellipsis != "…" {
		t.Fatalf("Ellipsis = %q, esperava …", theme.Ellipsis)
	}
}

func TestThemeNormalizadoPreencheGlifos(t *testing.T) {
	theme := Theme{}.normalized()
	if theme.FullChar == "" || theme.EmptyChar == "" || theme.Ellipsis == "" {
		t.Fatal("glifos vazios não foram preenchidos")
	}
	for _, s := range []State{Pending, Running, Done, Failed, Cancelled} {
		if theme.symbol(s) == "" {
			t.Fatalf("símbolo vazio para o estado %d", s)
		}
	}
}

func TestThemeSimboloPorEstado(t *testing.T) {
	def := DefaultTheme()
	tests := []struct {
		state State
		want  string
	}{
		{Pending, def.PendingSymbol},
		{Running, def.RunningSymbol},
		{Done, def.DoneSymbol},
		{Failed, def.FailedSymbol},
		{Cancelled, def.CancelledSymbol},
		{State(99), def.PendingSymbol},
	}
	for _, tt := range tests {
		if got := def.symbol(tt.state); got != tt.want {
			t.Fatalf("symbol(%d) = %q, quer %q", tt.state, got, tt.want)
		}
	}
}

func TestTemaASCIiSimboloPorEstado(t *testing.T) {
	theme := ASCIiTheme()
	tests := []struct {
		state State
		want  string
	}{
		{Pending, "."},
		{Running, ">"},
		{Done, "+"},
		{Failed, "x"},
		{Cancelled, "-"},
	}
	for _, tt := range tests {
		if got := theme.symbol(tt.state); got != tt.want {
			t.Fatalf("symbol(%d) = %q, quer %q", tt.state, got, tt.want)
		}
	}
}
