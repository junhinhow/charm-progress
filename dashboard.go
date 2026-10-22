package progress

import (
	"math"
	"time"
)

// Dashboard mantém um conjunto ordenado de tarefas e calcula o progresso
// agregado. A ordem de inserção é preservada e usada em toda a renderização.
//
// O Dashboard não é seguro para uso concorrente; use uma única goroutine ou
// proteja o acesso com um mutex.
type Dashboard struct {
	tasks      []Task
	index      map[string]int
	theme      Theme
	barWidth   int
	showETA    bool
	showCounts bool
	maxVisible int

	startedAt time.Time
	now       func() time.Time
}

// Option altera a configuração de um Dashboard durante New.
type Option func(*Dashboard)

// WithTheme define o tema visual do dashboard.
func WithTheme(t Theme) Option {
	return func(d *Dashboard) {
		d.theme = t.normalized()
	}
}

// WithBarWidth define a largura desejada da barra em colunas.
//
// O valor é limitado ao intervalo [5, 60]; larguras menores podem ser usadas
// na renderização quando o espaço disponível não comportar o valor desejado.
func WithBarWidth(n int) Option {
	return func(d *Dashboard) {
		d.barWidth = clamp(n, 5, 60)
	}
}

// WithShowETA liga ou desliga a exibição da estimativa de término no resumo.
func WithShowETA(b bool) Option {
	return func(d *Dashboard) {
		d.showETA = b
	}
}

// WithShowCounts liga ou desliga a exibição de contagens numéricas.
//
// Em larguras muito estreitas as contagens podem ser omitidas para preservar
// a barra e o rótulo.
func WithShowCounts(b bool) Option {
	return func(d *Dashboard) {
		d.showCounts = b
	}
}

// WithMaxVisible limita quantas tarefas são renderizadas por View.
//
// O valor mínimo é 1; quando há tarefas ocultas, View acrescenta uma linha
// com a quantidade restante.
func WithMaxVisible(n int) Option {
	return func(d *Dashboard) {
		if n < 1 {
			n = 1
		}
		d.maxVisible = n
	}
}

// New cria um dashboard vazio com o tema padrão, barra de 20 colunas,
// contagens e ETA ligados e sem limite de tarefas visíveis.
func New(opts ...Option) *Dashboard {
	d := &Dashboard{
		index:      make(map[string]int),
		theme:      DefaultTheme(),
		barWidth:   20,
		showETA:    true,
		showCounts: true,
		now:        time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}
	return d
}

// Add insere uma tarefa no fim do dashboard.
//
// Se já existir uma tarefa com o mesmo ID, ela é substituída na mesma posição.
func (d *Dashboard) Add(t Task) *Dashboard {
	if d.index == nil {
		d.index = make(map[string]int)
	}
	if t.Current < 0 {
		t.Current = 0
	}
	if i, ok := d.index[t.ID]; ok {
		d.tasks[i] = t
		return d
	}
	d.index[t.ID] = len(d.tasks)
	d.tasks = append(d.tasks, t)
	return d
}

// Update define o valor atual da tarefa identificada por id.
//
// Valores negativos são tratados como zero. Uma tarefa em Pending passa a
// Running; tarefas em estados finais mantêm o estado. IDs desconhecidos são
// ignorados. O primeiro Update marca o início da medição de ETA.
func (d *Dashboard) Update(id string, current int64) *Dashboard {
	i, ok := d.index[id]
	if !ok {
		return d
	}
	if current < 0 {
		current = 0
	}
	t := &d.tasks[i]
	t.Current = current
	if t.State == Pending {
		t.State = Running
	}
	if d.startedAt.IsZero() {
		d.startedAt = d.clock()
	}
	return d
}

// Advance soma delta ao valor atual da tarefa identificada por id.
//
// O resultado nunca fica negativo. IDs desconhecidos são ignorados.
func (d *Dashboard) Advance(id string, delta int64) *Dashboard {
	i, ok := d.index[id]
	if !ok {
		return d
	}
	current := addSaturated(d.tasks[i].Current, delta)
	if current < 0 {
		current = 0
	}
	return d.Update(id, current)
}

// Done marca a tarefa como concluída e, quando Total é positivo, define
// Current igual a Total. O erro anterior é limpo.
func (d *Dashboard) Done(id string) *Dashboard {
	i, ok := d.index[id]
	if !ok {
		return d
	}
	t := &d.tasks[i]
	if t.Total > 0 {
		t.Current = t.Total
	}
	t.State = Done
	t.Err = nil
	return d
}

// Fail marca a tarefa como falha e guarda o erro informado. Current é
// preservado para indicar até onde o trabalho chegou.
func (d *Dashboard) Fail(id string, err error) *Dashboard {
	i, ok := d.index[id]
	if !ok {
		return d
	}
	t := &d.tasks[i]
	t.State = Failed
	t.Err = err
	return d
}

// Cancel marca a tarefa como cancelada, preservando Current e limpando o erro.
func (d *Dashboard) Cancel(id string) *Dashboard {
	i, ok := d.index[id]
	if !ok {
		return d
	}
	t := &d.tasks[i]
	t.State = Cancelled
	t.Err = nil
	return d
}

// Remove exclui a tarefa identificada por id e informa se ela existia.
func (d *Dashboard) Remove(id string) bool {
	i, ok := d.index[id]
	if !ok {
		return false
	}
	d.tasks = append(d.tasks[:i], d.tasks[i+1:]...)
	delete(d.index, id)
	for j := i; j < len(d.tasks); j++ {
		d.index[d.tasks[j].ID] = j
	}
	return true
}

// Task devolve uma cópia da tarefa identificada por id.
func (d *Dashboard) Task(id string) (Task, bool) {
	i, ok := d.index[id]
	if !ok {
		return Task{}, false
	}
	return d.tasks[i], true
}

// Tasks devolve uma cópia das tarefas na ordem de inserção.
func (d *Dashboard) Tasks() []Task {
	out := make([]Task, len(d.tasks))
	copy(out, d.tasks)
	return out
}

// Overall devolve o progresso agregado entre 0 e 100.
//
// O cálculo é ponderado por Total. Tarefas com Total menor ou igual a zero
// não contribuem para a ponderação; se todas estiverem nessa condição, o
// resultado é a média simples das tarefas concluídas (Done vale 100 e as
// demais valem 0).
func (d *Dashboard) Overall() float64 {
	if len(d.tasks) == 0 {
		return 0
	}

	var weight, weighted float64
	for _, t := range d.tasks {
		if t.Total <= 0 {
			continue
		}
		current := t.Current
		if current < 0 {
			current = 0
		}
		if current > t.Total {
			current = t.Total
		}
		weight += float64(t.Total)
		weighted += float64(current)
	}
	if weight > 0 {
		p := weighted / weight * 100
		return clampFloat(p, 0, 100)
	}

	var done float64
	for _, t := range d.tasks {
		if t.State == Done {
			done++
		}
	}
	return done / float64(len(d.tasks)) * 100
}

// ETA estima o tempo restante com base nas tarefas em Running com Total
// positivo e no tempo decorrido desde o primeiro Update.
//
// Retorna zero quando não há tarefas em execução, quando nenhum progresso foi
// registrado desde o início ou quando now é anterior ao início. O relógio é
// injetado por now para permitir testes determinísticos.
func (d *Dashboard) ETA(now time.Time) time.Duration {
	if d.startedAt.IsZero() || now.Before(d.startedAt) {
		return 0
	}
	elapsed := now.Sub(d.startedAt)
	if elapsed <= 0 {
		return 0
	}

	var processed, remaining int64
	running := 0
	for _, t := range d.tasks {
		if t.State != Running || t.Total <= 0 {
			continue
		}
		running++
		current := t.Current
		if current < 0 {
			current = 0
		}
		if current > t.Total {
			current = t.Total
		}
		processed = addSaturated(processed, current)
		remaining = addSaturated(remaining, t.Remaining())
	}
	if running == 0 || processed <= 0 || remaining <= 0 {
		return 0
	}

	rate := float64(processed) / elapsed.Seconds()
	if rate <= 0 {
		return 0
	}
	seconds := float64(remaining) / rate
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 {
		return 0
	}
	if seconds >= float64(math.MaxInt64)/float64(time.Second) {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(seconds * float64(time.Second))
}

// Active devolve quantas tarefas estão em Running.
func (d *Dashboard) Active() int {
	n := 0
	for _, t := range d.tasks {
		if t.State == Running {
			n++
		}
	}
	return n
}

// Failed devolve quantas tarefas estão em Failed.
func (d *Dashboard) Failed() int {
	n := 0
	for _, t := range d.tasks {
		if t.State == Failed {
			n++
		}
	}
	return n
}

// Finished informa se todas as tarefas chegaram a um estado final.
//
// Um dashboard vazio não é considerado finalizado.
func (d *Dashboard) Finished() bool {
	if len(d.tasks) == 0 {
		return false
	}
	for _, t := range d.tasks {
		if !t.State.terminal() {
			return false
		}
	}
	return true
}

// clock devolve o relógio configurado, com fallback para time.Now.
func (d *Dashboard) clock() time.Time {
	if d.now == nil {
		return time.Now()
	}
	return d.now()
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func addSaturated(a, b int64) int64 {
	if b > 0 && a > math.MaxInt64-b {
		return math.MaxInt64
	}
	if b < 0 && a < math.MinInt64-b {
		return math.MinInt64
	}
	return a + b
}
