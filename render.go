package progress

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// View renderiza o dashboard completo dentro da largura informada.
//
// Cada linha tem no máximo width colunas de exibição, medidas com
// lipgloss.Width. Larguras menores ou iguais a zero, ou um dashboard sem
// tarefas, produzem string vazia. Com WithMaxVisible, apenas as primeiras N
// tarefas aparecem, seguidas por uma linha com a quantidade oculta.
func (d *Dashboard) View(width int) string {
	if width <= 0 || len(d.tasks) == 0 {
		return ""
	}

	visible := d.tasks
	hidden := 0
	if d.maxVisible > 0 && len(visible) > d.maxVisible {
		hidden = len(visible) - d.maxVisible
		visible = visible[:d.maxVisible]
	}

	layout := d.layoutFor(visible, width)
	lines := make([]string, 0, len(visible)+2)
	for _, t := range visible {
		lines = append(lines, d.renderTask(t, width, layout))
	}
	if hidden > 0 {
		word := "tarefas"
		if hidden == 1 {
			word = "tarefa"
		}
		muted := d.theme.Muted.Render(fmt.Sprintf("%s +%d %s", d.theme.Ellipsis, hidden, word))
		lines = append(lines, ansi.Truncate(muted, width, ""))
	}
	lines = append(lines, d.summary(width))
	return strings.Join(lines, "\n")
}

// ViewTask renderiza uma única tarefa em uma linha com a largura informada.
//
// A tarefa não precisa pertencer ao dashboard; os estilos e opções do
// dashboard são aplicados. Larguras menores ou iguais a zero produzem string
// vazia.
func (d *Dashboard) ViewTask(t Task, width int) string {
	if width <= 0 {
		return ""
	}
	return d.renderTask(t, width, d.layoutFor([]Task{t}, width))
}

// lineLayout guarda as larguras comuns para alinhar barras e sufixos.
type lineLayout struct {
	prefixW int
	suffixW int
	labelW  int
	barW    int
	counts  bool
}

// layoutFor calcula uma única largura de rótulo e de barra para todas as
// tarefas visíveis, mantendo as linhas alinhadas.
//
// Quando as contagens numéricas impedem a barra de aparecer, o layout é
// recalculado sem elas: em terminais estreitos a barra e o rótulo têm
// prioridade sobre os números.
func (d *Dashboard) layoutFor(tasks []Task, width int) lineLayout {
	layout := d.computeLayout(tasks, width, d.showCounts)
	if d.showCounts && layout.barW == 0 {
		alternative := d.computeLayout(tasks, width, false)
		if alternative.barW > 0 || alternative.labelW > layout.labelW {
			return alternative
		}
	}
	return layout
}

// computeLayout calcula o layout com ou sem contagens.
func (d *Dashboard) computeLayout(tasks []Task, width int, counts bool) lineLayout {
	layout := lineLayout{counts: counts}
	for _, t := range tasks {
		if w := lipgloss.Width(d.prefix(t)); w > layout.prefixW {
			layout.prefixW = w
		}
		if w := lipgloss.Width(d.suffix(t, counts)); w > layout.suffixW {
			layout.suffixW = w
		}
	}

	avail := width - layout.prefixW - layout.suffixW - 1
	if avail < 1 {
		return layout
	}

	const labelMin = 12
	barW := d.barWidth
	if avail < barW+1+labelMin {
		barW = avail - 1 - labelMin
	}
	if barW < 5 {
		barW = 0
	}

	layout.barW = barW
	if barW > 0 {
		layout.labelW = avail - barW - 1
	} else {
		layout.labelW = avail
	}
	if layout.labelW < 0 {
		layout.labelW = 0
	}
	return layout
}

// renderTask monta uma linha já truncada para width.
func (d *Dashboard) renderTask(t Task, width int, layout lineLayout) string {
	label := sanitizeLabel(t.Label)
	if label == "" {
		label = sanitizeLabel(t.ID)
	}
	if label == "" {
		label = "tarefa"
	}

	prefix := d.prefix(t)
	suffix := d.suffix(t, layout.counts)

	if layout.labelW <= 0 {
		// Espaço insuficiente para rótulo e barra: preserva o estado e o
		// percentual, que são a informação essencial da linha.
		line := prefix + d.theme.Muted.Render(formatPercent(t.Percent()))
		return ansi.Truncate(line, width, "")
	}

	prefix += strings.Repeat(" ", layout.prefixW-lipgloss.Width(prefix))
	suffix = strings.Repeat(" ", layout.suffixW-lipgloss.Width(suffix)) + suffix

	text := truncate(label, layout.labelW, d.theme.Ellipsis)
	text = d.theme.Label.Render(text)
	if pad := layout.labelW - lipgloss.Width(text); pad > 0 {
		text += strings.Repeat(" ", pad)
	}

	if layout.barW > 0 {
		text += " " + d.bar(t.Percent(), layout.barW) + " "
	} else {
		text += " "
	}
	return ansi.Truncate(prefix+text+suffix, width, "")
}

// prefix devolve o símbolo estilizado do estado seguido de um espaço.
func (d *Dashboard) prefix(t Task) string {
	return d.theme.stateStyle(t.State).Render(d.theme.symbol(t.State)) + " "
}

// suffix devolve contagem e percentual, quando habilitados.
func (d *Dashboard) suffix(t Task, counts bool) string {
	percent := formatPercent(t.Percent())
	if !counts {
		return d.theme.Muted.Render(percent)
	}
	return d.theme.Muted.Render(formatCounts(t) + "  " + percent)
}

// summary monta a linha de total do dashboard.
func (d *Dashboard) summary(width int) string {
	label := d.theme.Label.Render("total")
	percent := d.theme.Muted.Render(formatPercent(d.Overall()))

	var extra string
	if d.showCounts {
		done := 0
		for _, t := range d.tasks {
			if t.State == Done {
				done++
			}
		}
		word := "tarefas"
		if len(d.tasks) == 1 {
			word = "tarefa"
		}
		extra += "  " + d.theme.Muted.Render(fmt.Sprintf("%d/%d %s", done, len(d.tasks), word))
	}
	if d.showETA {
		if eta := d.ETA(d.clock()); eta > 0 {
			extra += "  " + d.theme.Muted.Render("ETA "+formatDuration(eta))
		}
	}

	barW := 0
	if width >= 32 {
		used := lipgloss.Width(label) + 1 + lipgloss.Width(percent) + lipgloss.Width(extra)
		if room := width - used - 1; room >= 5 {
			barW = d.barWidth
			if barW > room {
				barW = room
			}
		}
	}

	line := label
	if barW >= 5 {
		line += " " + d.bar(d.Overall(), barW)
	}
	line += " " + percent
	line = appendIfFits(line, extra, width)
	return ansi.Truncate(line, width, "")
}

// bar desenha uma barra de progresso com exatamente cells colunas.
func (d *Dashboard) bar(percent float64, cells int) string {
	if cells <= 0 {
		return ""
	}
	percent = clampFloat(percent, 0, 100)
	filled := int(math.Floor(float64(cells) * percent / 100))
	if percent >= 100 {
		filled = cells
	}
	if filled < 0 {
		filled = 0
	}
	if filled > cells {
		filled = cells
	}

	full := d.theme.BarFull.Render(repeatToWidth(d.theme.FullChar, filled))
	empty := d.theme.BarEmpty.Render(repeatToWidth(d.theme.EmptyChar, cells-filled))
	return full + empty
}

// formatPercent devolve o percentual arredondado com quatro colunas fixas.
func formatPercent(p float64) string {
	return fmt.Sprintf("%3.0f%%", clampFloat(p, 0, 100))
}

// formatCounts devolve a contagem atual, com o total quando conhecido.
func formatCounts(t Task) string {
	current := t.Current
	if current < 0 {
		current = 0
	}
	if t.Total <= 0 {
		return fmt.Sprintf("%d", current)
	}
	return fmt.Sprintf("%d/%d", current, t.Total)
}

// formatDuration apresenta a duração de forma compacta e estável.
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	if d < time.Second {
		return "<1s"
	}
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		minutes := int(d / time.Minute)
		seconds := int((d % time.Minute) / time.Second)
		return fmt.Sprintf("%dm%02ds", minutes, seconds)
	}
	if d < 24*time.Hour {
		hours := int(d / time.Hour)
		minutes := int((d % time.Hour) / time.Minute)
		return fmt.Sprintf("%dh%02dm", hours, minutes)
	}
	days := int(d / (24 * time.Hour))
	hours := int((d % (24 * time.Hour)) / time.Hour)
	return fmt.Sprintf("%dd%02dh", days, hours)
}

// appendIfFits acrescenta extra somente se a linha continuar dentro de width.
func appendIfFits(line, extra string, width int) string {
	if lipgloss.Width(line)+lipgloss.Width(extra) > width {
		return line
	}
	return line + extra
}

// truncate corta s em width colunas, usando tail apenas quando ele cabe.
func truncate(s string, width int, tail string) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if lipgloss.Width(tail) >= width {
		return ansi.Truncate(s, width, "")
	}
	return ansi.Truncate(s, width, tail)
}

// repeatToWidth repete ch até preencher exatamente cells colunas.
func repeatToWidth(ch string, cells int) string {
	if cells <= 0 {
		return ""
	}
	w := lipgloss.Width(ch)
	if w <= 0 {
		ch = " "
		w = 1
	}
	n := cells / w
	rest := cells - n*w
	var b strings.Builder
	b.Grow(cells)
	for i := 0; i < n; i++ {
		b.WriteString(ch)
	}
	if rest > 0 {
		b.WriteString(strings.Repeat(" ", rest))
	}
	return b.String()
}

// sanitizeLabel remove quebras de linha e caracteres de controle para que
// cada tarefa ocupe exatamente uma linha.
func sanitizeLabel(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsControl(r) {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}
