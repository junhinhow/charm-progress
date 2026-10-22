package progress

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func sampleDashboard(opts ...Option) *Dashboard {
	options := append([]Option{WithShowETA(false)}, opts...)
	d := New(options...)
	d.Add(Task{ID: "download", Label: "Baixar arquivos", Total: 200, Current: 50, State: Running})
	d.Add(Task{ID: "convert", Label: "Converter imagens", Total: 100, Current: 100, State: Done})
	d.Add(Task{ID: "upload", Label: "Enviar resultado", Total: 80, Current: 20, State: Failed})
	d.Add(Task{ID: "cleanup", Label: "Limpar temporários", Total: 40, Current: 10, State: Cancelled})
	d.Add(Task{ID: "pending", Label: "Publicar pacote", State: Pending})
	return d
}

func TestViewRespeitaLargura(t *testing.T) {
	d := sampleDashboard(WithBarWidth(30))
	d.Add(Task{ID: "long", Label: "rótulo muito comprido com acentuação e detalhes", Total: 1000, Current: 999, State: Running})

	for _, width := range []int{1, 20, 40, 80, 0, -5} {
		out := d.View(width)
		if width <= 0 {
			if out != "" {
				t.Fatalf("View(%d) = %q, quer string vazia", width, out)
			}
			continue
		}
		if out == "" {
			t.Fatalf("View(%d) vazio", width)
		}
		for i, line := range strings.Split(out, "\n") {
			if got := lipgloss.Width(line); got > width {
				t.Fatalf("View(%d) linha %d tem largura %d: %q", width, i, got, line)
			}
		}
	}
}

func TestViewDeterministico(t *testing.T) {
	d := sampleDashboard()
	first := d.View(60)
	for i := 0; i < 100; i++ {
		if got := d.View(60); got != first {
			t.Fatalf("View mudou na chamada %d:\nprimeira: %q\natual:    %q", i+1, first, got)
		}
	}
}

func TestViewMantemOrdemDeInsercao(t *testing.T) {
	d := New(WithTheme(ASCIiTheme()), WithShowETA(false), WithShowCounts(false))
	d.Add(Task{ID: "a", Label: "alfa", Total: 10, Current: 5})
	d.Add(Task{ID: "b", Label: "beta", Total: 10, Current: 5})
	d.Add(Task{ID: "c", Label: "gama", Total: 10, Current: 5})

	lines := strings.Split(d.View(60), "\n")
	for i, want := range []string{"alfa", "beta", "gama"} {
		if !strings.Contains(lines[i], want) {
			t.Fatalf("linha %d = %q, quer conter %q", i, lines[i], want)
		}
	}
}

func TestWithMaxVisible(t *testing.T) {
	d := New(WithShowETA(false), WithMaxVisible(2))
	for i := 1; i <= 5; i++ {
		d.Add(Task{ID: string(rune('0' + i)), Label: "tarefa " + string(rune('0'+i)), Total: 10})
	}

	out := d.View(80)
	lines := strings.Split(out, "\n")
	if len(lines) != 4 {
		t.Fatalf("View com 2 visíveis tem %d linhas, quer 4:\n%s", len(lines), out)
	}
	if !strings.Contains(out, "… +3 tarefas") {
		t.Fatalf("linha de ocultas ausente:\n%s", out)
	}
	if strings.Contains(out, "tarefa 3") {
		t.Fatalf("tarefa oculta foi renderizada:\n%s", out)
	}

	d = New(WithShowETA(false), WithMaxVisible(4))
	for i := 1; i <= 5; i++ {
		d.Add(Task{ID: string(rune('0' + i)), Label: "tarefa " + string(rune('0'+i)), Total: 10})
	}
	if out := d.View(80); !strings.Contains(out, "… +1 tarefa") {
		t.Fatalf("singular incorreto:\n%s", out)
	}

	d = New(WithTheme(ASCIiTheme()), WithShowETA(false), WithMaxVisible(2))
	for i := 1; i <= 5; i++ {
		d.Add(Task{ID: string(rune('0' + i)), Label: "tarefa " + string(rune('0'+i)), Total: 10})
	}
	if out := d.View(80); !strings.Contains(out, "... +3 tarefas") {
		t.Fatalf("elipse ASCII ausente:\n%s", out)
	}
}

func TestTodosOsEstadosAparecem(t *testing.T) {
	theme := DefaultTheme()
	d := New(WithShowETA(false))
	states := []State{Pending, Running, Done, Failed, Cancelled}
	for i, state := range states {
		d.Add(Task{
			ID:      string(rune('a' + i)),
			Label:   "tarefa",
			Total:   100,
			Current: int64(i * 25),
			State:   state,
		})
	}

	out := d.View(100)
	lines := strings.Split(out, "\n")
	for i, state := range states {
		symbol := theme.symbol(state)
		if !strings.Contains(out, symbol) {
			t.Fatalf("símbolo %q do estado %d não apareceu:\n%s", symbol, state, out)
		}
		if !strings.HasPrefix(ansi.Strip(lines[i]), symbol+" ") {
			t.Fatalf("linha %d = %q, quer começar com %q", i, lines[i], symbol+" ")
		}
	}
}

func TestASCIiThemeNaoEmiteNaoASCII(t *testing.T) {
	d := New(WithTheme(ASCIiTheme()), WithShowETA(false))
	states := []State{Pending, Running, Done, Failed, Cancelled}
	for i, state := range states {
		d.Add(Task{
			ID:      string(rune('a' + i)),
			Label:   "task " + string(rune('a'+i)),
			Total:   100,
			Current: int64(i * 20),
			State:   state,
		})
	}

	out := d.View(80)
	for _, r := range out {
		if r > 127 {
			t.Fatalf("saída ASCII contém rune %q (U+%04X):\n%s", r, r, out)
		}
	}
}

func TestViewTask(t *testing.T) {
	d := New(WithTheme(ASCIiTheme()), WithShowETA(false), WithShowCounts(false))
	task := Task{ID: "a", Label: "um rótulo longo para truncar", Total: 100, Current: 50, State: Running}

	out := d.ViewTask(task, 30)
	if strings.Contains(out, "\n") {
		t.Fatalf("ViewTask devolveu mais de uma linha: %q", out)
	}
	if got := lipgloss.Width(out); got > 30 {
		t.Fatalf("ViewTask excedeu a largura: %d > 30 (%q)", got, out)
	}
	if !strings.HasPrefix(out, "> ") {
		t.Fatalf("ViewTask = %q, quer começar com o símbolo Running", out)
	}
	if out := d.ViewTask(task, 0); out != "" {
		t.Fatalf("ViewTask(0) = %q, quer vazio", out)
	}
}

func TestViewTaskAlinhamentoASCII(t *testing.T) {
	d := New(WithTheme(ASCIiTheme()), WithShowETA(false), WithShowCounts(false))
	task := Task{ID: "a", Label: "abc", Total: 10, Current: 5, State: Running}

	const width = 40
	want := "> abc" + strings.Repeat(" ", 10) + "##########----------" + "  50%"
	got := d.ViewTask(task, width)
	if got != want {
		t.Fatalf("ViewTask = %q\nquer         %q", got, want)
	}
	if lipgloss.Width(got) != width {
		t.Fatalf("largura = %d, quer %d", lipgloss.Width(got), width)
	}
}

func TestViewVazio(t *testing.T) {
	if out := New().View(80); out != "" {
		t.Fatalf("View de dashboard vazio = %q, quer vazio", out)
	}
}

func TestViewLarguraComUnicodeLarga(t *testing.T) {
	d := New(WithShowETA(false))
	d.Add(Task{ID: "a", Label: "下载文件😀 muito comprido", Total: 100, Current: 50, State: Running})
	d.Add(Task{ID: "b", Label: "日本語のラベル", Total: 100, Current: 25, State: Running})

	for _, width := range []int{8, 16, 24, 40} {
		out := d.View(width)
		for i, line := range strings.Split(out, "\n") {
			if got := lipgloss.Width(line); got > width {
				t.Fatalf("View(%d) linha %d tem largura %d: %q", width, i, got, line)
			}
		}
	}
}

func TestViewSanitizaLabel(t *testing.T) {
	d := New(WithShowETA(false))
	d.Add(Task{ID: "a", Label: "primeira\nsegunda\tlinha", Total: 10, Current: 5})
	out := d.View(60)
	if strings.Contains(out, "\t") {
		t.Fatalf("label com tab não foi sanitizado: %q", out)
	}
	if got := len(strings.Split(out, "\n")); got != 2 {
		t.Fatalf("View gerou %d linhas, quer 2 (tarefa + total):\n%s", got, out)
	}
	if !strings.Contains(out, "primeira segunda linha") {
		t.Fatalf("label sanitizado ausente:\n%s", out)
	}
}

func TestViewExibeETAComRelogioInjetavel(t *testing.T) {
	base := time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)
	now := base
	d := New()
	d.now = func() time.Time { return now }
	d.Add(Task{ID: "a", Label: "download", Total: 100, State: Running})
	d.Update("a", 20)
	now = base.Add(10 * time.Second)

	out := d.View(100)
	if !strings.Contains(out, "ETA 40s") {
		t.Fatalf("ETA ausente no resumo:\n%s", out)
	}

	d = New(WithShowETA(false))
	d.now = func() time.Time { return now }
	d.Add(Task{ID: "a", Label: "download", Total: 100, State: Running})
	d.Update("a", 20)
	if out := d.View(100); strings.Contains(out, "ETA") {
		t.Fatalf("ETA exibida com WithShowETA(false):\n%s", out)
	}
}

func TestViewSemContagens(t *testing.T) {
	d := New(WithShowETA(false), WithShowCounts(false))
	d.Add(Task{ID: "a", Label: "download", Total: 100, Current: 50})
	out := d.View(80)
	if strings.Contains(out, "50/100") {
		t.Fatalf("contagem exibida com WithShowCounts(false):\n%s", out)
	}
	if !strings.Contains(out, "50%") {
		t.Fatalf("percentual ausente:\n%s", out)
	}
}
