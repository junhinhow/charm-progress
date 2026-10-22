package progress

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestAddPreservaOrdemESubstituiDuplicado(t *testing.T) {
	d := New()
	d.Add(Task{ID: "a", Label: "primeira", Total: 10})
	d.Add(Task{ID: "b", Label: "segunda", Total: 20})
	d.Add(Task{ID: "c", Label: "terceira", Total: 30})
	d.Add(Task{ID: "b", Label: "substituída", Total: 99, Current: 5})

	tasks := d.Tasks()
	if len(tasks) != 3 {
		t.Fatalf("len(Tasks()) = %d, quer 3", len(tasks))
	}
	wantIDs := []string{"a", "b", "c"}
	for i, want := range wantIDs {
		if tasks[i].ID != want {
			t.Fatalf("Tasks()[%d].ID = %q, quer %q", i, tasks[i].ID, want)
		}
	}
	if tasks[1].Label != "substituída" || tasks[1].Total != 99 || tasks[1].Current != 5 {
		t.Fatalf("tarefa duplicada não foi substituída: %+v", tasks[1])
	}
}

func TestTasksDevolveCopia(t *testing.T) {
	d := New()
	d.Add(Task{ID: "a", Label: "original", Total: 10})
	got := d.Tasks()
	got[0].Label = "alterado"
	got = append(got, Task{ID: "b"})

	if len(d.Tasks()) != 1 {
		t.Fatalf("Tasks() expôs o slice interno")
	}
	if task, _ := d.Task("a"); task.Label != "original" {
		t.Fatalf("Task(a).Label = %q, quer original", task.Label)
	}
	if _, ok := d.Task("b"); ok {
		t.Fatal("Task(b) encontrou tarefa inexistente")
	}
}

func TestAddNormalizaCurrentNegativo(t *testing.T) {
	d := New()
	d.Add(Task{ID: "a", Total: 10, Current: -5})
	if task, _ := d.Task("a"); task.Current != 0 {
		t.Fatalf("Current = %d, quer 0", task.Current)
	}
}

func TestUpdateAdvanceEDone(t *testing.T) {
	d := New()
	d.Add(Task{ID: "a", Label: "a", Total: 100})

	d.Update("a", 25)
	task, _ := d.Task("a")
	if task.State != Running || task.Current != 25 {
		t.Fatalf("após Update: %+v, quer Running/25", task)
	}

	d.Update("a", -10)
	if task, _ = d.Task("a"); task.Current != 0 {
		t.Fatalf("Update negativo deixou Current = %d", task.Current)
	}

	d.Advance("a", 40)
	if task, _ = d.Task("a"); task.Current != 40 {
		t.Fatalf("após Advance: Current = %d, quer 40", task.Current)
	}
	d.Advance("a", -100)
	if task, _ = d.Task("a"); task.Current != 0 {
		t.Fatalf("Advance negativo deixou Current = %d", task.Current)
	}

	d.Update("a", 30)
	d.Done("a")
	if task, _ = d.Task("a"); task.State != Done || task.Current != 100 || task.Err != nil {
		t.Fatalf("após Done: %+v, quer Done/100/sem erro", task)
	}

	d.Update("a", 10)
	if task, _ = d.Task("a"); task.State != Done || task.Current != 10 {
		t.Fatalf("Update reabriu tarefa concluída: %+v", task)
	}
}

func TestAdvanceSaturaSemEstouro(t *testing.T) {
	d := New()
	d.Add(Task{ID: "a", Total: math.MaxInt64})
	d.Update("a", math.MaxInt64-5)
	d.Advance("a", 100)

	task, _ := d.Task("a")
	if task.Current != math.MaxInt64 {
		t.Fatalf("Current = %d, quer MaxInt64", task.Current)
	}
}

func TestDoneComTotalZeroPreservaCurrent(t *testing.T) {
	d := New()
	d.Add(Task{ID: "a", Total: 0, Current: 7, State: Running})
	d.Done("a")
	task, _ := d.Task("a")
	if task.State != Done || task.Current != 7 {
		t.Fatalf("Done com Total 0: %+v, quer Done/7", task)
	}
}

func TestFailECancel(t *testing.T) {
	d := New()
	d.Add(Task{ID: "a", Total: 100, Current: 40, State: Running})
	sentinel := errors.New("falha de rede")
	d.Fail("a", sentinel)
	task, _ := d.Task("a")
	if task.State != Failed || task.Current != 40 || !errors.Is(task.Err, sentinel) {
		t.Fatalf("após Fail: %+v", task)
	}

	d.Cancel("a")
	task, _ = d.Task("a")
	if task.State != Cancelled || task.Current != 40 || task.Err != nil {
		t.Fatalf("após Cancel: %+v", task)
	}
}

func TestOperacoesComIDDesconhecidoSaoIgnoradas(t *testing.T) {
	d := New()
	d.Update("x", 1)
	d.Advance("x", 1)
	d.Done("x")
	d.Fail("x", errors.New("x"))
	d.Cancel("x")

	if len(d.Tasks()) != 0 {
		t.Fatalf("operações criaram tarefas: %+v", d.Tasks())
	}
	if d.Finished() {
		t.Fatal("dashboard vazio não deveria estar finalizado")
	}
}

func TestRemoveReindexaOrdem(t *testing.T) {
	d := New()
	d.Add(Task{ID: "a"})
	d.Add(Task{ID: "b"})
	d.Add(Task{ID: "c"})

	if !d.Remove("b") {
		t.Fatal("Remove(b) = false, quer true")
	}
	if d.Remove("b") {
		t.Fatal("Remove(b) repetido = true, quer false")
	}
	if _, ok := d.Task("b"); ok {
		t.Fatal("Task(b) ainda existe após Remove")
	}
	d.Add(Task{ID: "d"})

	want := []string{"a", "c", "d"}
	tasks := d.Tasks()
	if len(tasks) != len(want) {
		t.Fatalf("len(Tasks()) = %d, quer %d", len(tasks), len(want))
	}
	for i, id := range want {
		if tasks[i].ID != id {
			t.Fatalf("Tasks()[%d].ID = %q, quer %q", i, tasks[i].ID, id)
		}
		if _, ok := d.Task(id); !ok {
			t.Fatalf("índice perdido para %q", id)
		}
	}
}

func TestOverall(t *testing.T) {
	tests := []struct {
		name  string
		tasks []Task
		want  float64
	}{
		{
			name: "ponderado por total",
			tasks: []Task{
				{Total: 100, Current: 50},
				{Total: 300, Current: 150},
			},
			want: 50,
		},
		{
			name:  "clampa current acima do total",
			tasks: []Task{{Total: 10, Current: 20}},
			want:  100,
		},
		{
			name: "todos os totais zero usa média simples",
			tasks: []Task{
				{State: Done},
				{State: Done},
				{State: Failed},
				{State: Pending},
			},
			want: 50,
		},
		{
			name: "total zero é ignorado quando há peso",
			tasks: []Task{
				{Total: 100, Current: 50},
				{State: Done},
			},
			want: 50,
		},
		{
			name: "dashboard vazio",
			want: 0,
		},
		{
			name: "totais negativos sem conclusão",
			tasks: []Task{
				{Total: -1},
				{Total: -2, State: Failed},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			for i, task := range tt.tasks {
				task.ID = string(rune('a' + i))
				d.Add(task)
			}
			got := d.Overall()
			if math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("Overall() = %v, quer %v", got, tt.want)
			}
		})
	}
}

func TestETAComRelogioInjetavel(t *testing.T) {
	base := time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)
	now := base
	d := New(WithShowETA(false))
	d.now = func() time.Time { return now }

	d.Add(Task{ID: "a", Total: 100, State: Running})
	d.Add(Task{ID: "b", Total: 50, State: Running})
	d.Add(Task{ID: "c", Total: 10, State: Pending})

	if got := d.ETA(base); got != 0 {
		t.Fatalf("ETA antes do primeiro Update = %v, quer 0", got)
	}

	d.Update("a", 20)
	now = base.Add(10 * time.Second)
	d.Update("b", 10)

	// 30 unidades em 10s = 3/s; faltam 80 + 40 = 120 unidades.
	if got := d.ETA(now); got != 40*time.Second {
		t.Fatalf("ETA = %v, quer 40s", got)
	}
	if got := d.ETA(base.Add(-time.Second)); got != 0 {
		t.Fatalf("ETA com now anterior = %v, quer 0", got)
	}

	d.Done("a")
	d.Done("b")
	d.Done("c")
	if got := d.ETA(now); got != 0 {
		t.Fatalf("ETA sem tarefas em execução = %v, quer 0", got)
	}

	// Tarefas sem total positivo não entram no cálculo do ETA.
	semTotal := New()
	semTotal.now = func() time.Time { return now }
	semTotal.Add(Task{ID: "x", State: Running})
	semTotal.Update("x", 50)
	if got := semTotal.ETA(now.Add(time.Minute)); got != 0 {
		t.Fatalf("ETA com Total zero = %v, quer 0", got)
	}
}

func TestETASemProgresso(t *testing.T) {
	base := time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)
	now := base
	d := New()
	d.now = func() time.Time { return now }
	d.Add(Task{ID: "a", Total: 100, State: Running})
	d.Update("a", 0)
	now = base.Add(time.Minute)
	if got := d.ETA(now); got != 0 {
		t.Fatalf("ETA sem progresso = %v, quer 0", got)
	}

	d.Update("a", 100)
	if got := d.ETA(now); got != 0 {
		t.Fatalf("ETA sem unidades restantes = %v, quer 0", got)
	}
}

func TestContadoresEFinished(t *testing.T) {
	d := New()
	if d.Finished() {
		t.Fatal("dashboard vazio não deveria estar finalizado")
	}
	d.Add(Task{ID: "a", State: Running})
	d.Add(Task{ID: "b", State: Pending})
	d.Add(Task{ID: "c", State: Failed})

	if got := d.Active(); got != 1 {
		t.Fatalf("Active() = %d, quer 1", got)
	}
	if got := d.Failed(); got != 1 {
		t.Fatalf("Failed() = %d, quer 1", got)
	}
	if d.Finished() {
		t.Fatal("dashboard com tarefas abertas não está finalizado")
	}

	d.Done("a")
	d.Cancel("b")
	if !d.Finished() {
		t.Fatal("dashboard com todos os estados finais deveria estar finalizado")
	}
}

func TestOpcoesFazemClamp(t *testing.T) {
	d := New(WithBarWidth(1), WithMaxVisible(0), WithShowETA(false), WithShowCounts(false))
	if d.barWidth != 5 {
		t.Fatalf("barWidth = %d, quer 5", d.barWidth)
	}
	if d.maxVisible != 1 {
		t.Fatalf("maxVisible = %d, quer 1", d.maxVisible)
	}
	if d.showETA || d.showCounts {
		t.Fatal("opções de exibição não foram aplicadas")
	}

	d = New(WithBarWidth(999))
	if d.barWidth != 60 {
		t.Fatalf("barWidth = %d, quer 60", d.barWidth)
	}

	d = New(nil, WithTheme(Theme{}))
	if d.theme.FullChar == "" {
		t.Fatal("tema vazio não foi normalizado")
	}
}
