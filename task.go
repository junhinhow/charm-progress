// Package progress implementa um dashboard de progresso para várias tarefas
// simultâneas, com barras individuais, progresso agregado e ETA.
package progress

// State representa a situação de uma tarefa no dashboard.
type State int

const (
	// Pending indica uma tarefa que ainda não começou.
	Pending State = iota
	// Running indica uma tarefa em execução.
	Running
	// Done indica uma tarefa concluída com sucesso.
	Done
	// Failed indica uma tarefa encerrada com erro.
	Failed
	// Cancelled indica uma tarefa cancelada antes do fim.
	Cancelled
)

// Task descreve uma tarefa acompanhada pelo dashboard.
//
// Total e Current usam a mesma unidade (bytes, itens, arquivos). Quando Total
// é menor ou igual a zero, a tarefa é tratada como indeterminada e Percent
// devolve zero.
type Task struct {
	ID      string
	Label   string
	Total   int64
	Current int64
	State   State
	Err     error
}

// Percent devolve o progresso da tarefa entre 0 e 100.
//
// O resultado é limitado ao intervalo fechado [0, 100], inclusive quando
// Current supera Total ou assume valor negativo. Total menor ou igual a zero
// produz zero, sem divisão por zero.
func (t Task) Percent() float64 {
	if t.Total <= 0 {
		return 0
	}
	current := t.Current
	if current <= 0 {
		return 0
	}
	if current >= t.Total {
		return 100
	}
	return float64(current) / float64(t.Total) * 100
}

// Remaining devolve quantas unidades ainda faltam para concluir a tarefa.
//
// O resultado nunca é negativo. Tarefas com Total menor ou igual a zero, ou
// já concluídas, devolvem zero.
func (t Task) Remaining() int64 {
	if t.Total <= 0 || t.Current >= t.Total {
		return 0
	}
	current := t.Current
	if current < 0 {
		current = 0
	}
	return t.Total - current
}

// terminal informa se o estado encerra a tarefa.
func (s State) terminal() bool {
	return s == Done || s == Failed || s == Cancelled
}
