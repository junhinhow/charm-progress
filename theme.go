package progress

import "charm.land/lipgloss/v2"

// Theme reúne os estilos e glifos usados na renderização do dashboard.
//
// Um tema parcial pode ser passado para WithTheme: campos de texto vazios
// (glifos e elipse) recebem o valor do tema padrão, enquanto estilos zerados
// são renderizados sem formatação. Para um tema sem caracteres Unicode use
// ASCIiTheme.
type Theme struct {
	// BarFull estiliza a parte preenchida da barra.
	BarFull lipgloss.Style
	// BarEmpty estiliza a parte vazia da barra.
	BarEmpty lipgloss.Style

	// Pending, Running, Done, Failed e Cancelled estilizam o símbolo de cada
	// estado da tarefa.
	Pending   lipgloss.Style
	Running   lipgloss.Style
	Done      lipgloss.Style
	Failed    lipgloss.Style
	Cancelled lipgloss.Style

	// Label estiliza o rótulo da tarefa.
	Label lipgloss.Style
	// Muted estiliza textos secundários, como contagens, percentuais e ETA.
	Muted lipgloss.Style

	// FullChar e EmptyChar compõem a barra de progresso.
	FullChar  string
	EmptyChar string

	// PendingSymbol, RunningSymbol, DoneSymbol, FailedSymbol e CancelledSymbol
	// identificam visualmente o estado de cada tarefa.
	PendingSymbol   string
	RunningSymbol   string
	DoneSymbol      string
	FailedSymbol    string
	CancelledSymbol string

	// Ellipsis indica texto truncado e tarefas ocultas.
	Ellipsis string
}

// DefaultTheme devolve o tema padrão, com glifos Unicode e cores para
// terminais que suportam cor.
func DefaultTheme() Theme {
	return Theme{
		BarFull:  lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")),
		BarEmpty: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),

		Pending:   lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Running:   lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),
		Done:      lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		Failed:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		Cancelled: lipgloss.NewStyle().Foreground(lipgloss.Color("244")),

		Label: lipgloss.NewStyle(),
		Muted: lipgloss.NewStyle().Foreground(lipgloss.Color("244")),

		FullChar:  "█",
		EmptyChar: "░",

		PendingSymbol:   "○",
		RunningSymbol:   "◐",
		DoneSymbol:      "✔",
		FailedSymbol:    "✖",
		CancelledSymbol: "⊘",

		Ellipsis: "…",
	}
}

// ASCIiTheme devolve um tema sem nenhum caractere não-ASCII, adequado a
// terminais simples, pipes e testes. A grafia do nome segue a API pública da
// biblioteca.
func ASCIiTheme() Theme {
	return Theme{
		BarFull:  lipgloss.NewStyle(),
		BarEmpty: lipgloss.NewStyle(),

		Pending:   lipgloss.NewStyle(),
		Running:   lipgloss.NewStyle(),
		Done:      lipgloss.NewStyle(),
		Failed:    lipgloss.NewStyle(),
		Cancelled: lipgloss.NewStyle(),

		Label: lipgloss.NewStyle(),
		Muted: lipgloss.NewStyle(),

		FullChar:  "#",
		EmptyChar: "-",

		PendingSymbol:   ".",
		RunningSymbol:   ">",
		DoneSymbol:      "+",
		FailedSymbol:    "x",
		CancelledSymbol: "-",

		Ellipsis: "...",
	}
}

// normalized preenche glifos vazios com os valores do tema padrão.
func (t Theme) normalized() Theme {
	def := DefaultTheme()
	if t.FullChar == "" {
		t.FullChar = def.FullChar
	}
	if t.EmptyChar == "" {
		t.EmptyChar = def.EmptyChar
	}
	if t.PendingSymbol == "" {
		t.PendingSymbol = def.PendingSymbol
	}
	if t.RunningSymbol == "" {
		t.RunningSymbol = def.RunningSymbol
	}
	if t.DoneSymbol == "" {
		t.DoneSymbol = def.DoneSymbol
	}
	if t.FailedSymbol == "" {
		t.FailedSymbol = def.FailedSymbol
	}
	if t.CancelledSymbol == "" {
		t.CancelledSymbol = def.CancelledSymbol
	}
	if t.Ellipsis == "" {
		t.Ellipsis = def.Ellipsis
	}
	return t
}

// symbol devolve o glifo do estado informado. Estados inválidos usam o glifo
// de Pending.
func (t Theme) symbol(s State) string {
	switch s {
	case Running:
		return t.RunningSymbol
	case Done:
		return t.DoneSymbol
	case Failed:
		return t.FailedSymbol
	case Cancelled:
		return t.CancelledSymbol
	default:
		return t.PendingSymbol
	}
}

// stateStyle devolve o estilo do estado informado. Estados inválidos usam o
// estilo de Pending.
func (t Theme) stateStyle(s State) lipgloss.Style {
	switch s {
	case Running:
		return t.Running
	case Done:
		return t.Done
	case Failed:
		return t.Failed
	case Cancelled:
		return t.Cancelled
	default:
		return t.Pending
	}
}
