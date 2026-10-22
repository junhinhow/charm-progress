package progress

import (
	"math"
	"testing"
)

func TestTaskPercent(t *testing.T) {
	tests := []struct {
		name string
		task Task
		want float64
	}{
		{"metade", Task{Total: 200, Current: 100}, 50},
		{"completa", Task{Total: 100, Current: 100}, 100},
		{"acima do total", Task{Total: 100, Current: 150}, 100},
		{"total zero", Task{Total: 0, Current: 10}, 0},
		{"total negativo", Task{Total: -10, Current: 3}, 0},
		{"current negativo", Task{Total: 100, Current: -10}, 0},
		{"tudo zero", Task{}, 0},
		{"arredondamento", Task{Total: 3, Current: 1}, 100.0 / 3.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.task.Percent()
			if math.IsNaN(got) {
				t.Fatalf("Percent() devolveu NaN")
			}
			if math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("Percent() = %v, quer %v", got, tt.want)
			}
			if got < 0 || got > 100 {
				t.Fatalf("Percent() = %v fora de [0, 100]", got)
			}
		})
	}
}

func TestTaskRemaining(t *testing.T) {
	tests := []struct {
		name string
		task Task
		want int64
	}{
		{"metade", Task{Total: 100, Current: 40}, 60},
		{"completa", Task{Total: 100, Current: 100}, 0},
		{"acima do total", Task{Total: 100, Current: 200}, 0},
		{"total zero", Task{Current: 10}, 0},
		{"total negativo", Task{Total: -1, Current: 0}, 0},
		{"current negativo", Task{Total: 10, Current: -5}, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.Remaining(); got != tt.want {
				t.Fatalf("Remaining() = %d, quer %d", got, tt.want)
			}
		})
	}
}

func TestStateTerminal(t *testing.T) {
	for _, s := range []State{Done, Failed, Cancelled} {
		if !s.terminal() {
			t.Fatalf("estado %d deveria ser final", s)
		}
	}
	for _, s := range []State{Pending, Running, State(99)} {
		if s.terminal() {
			t.Fatalf("estado %d não deveria ser final", s)
		}
	}
}
