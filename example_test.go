package progress_test

import (
	"fmt"

	"github.com/junhinhow/charm-progress"
)

// Example demonstra um lote de downloads em andamento.
func Example() {
	d := progress.New(
		progress.WithTheme(progress.ASCIiTheme()),
		progress.WithBarWidth(10),
		progress.WithShowETA(false),
	)
	d.Add(progress.Task{ID: "dl-1", Label: "baixar-1", Total: 100})
	d.Add(progress.Task{ID: "dl-2", Label: "baixar-2", Total: 100})
	d.Update("dl-1", 60)
	d.Done("dl-2")

	fmt.Println(d.View(46))
	// Output:
	// > baixar-1            ######----  60/100   60%
	// + baixar-2            ########## 100/100  100%
	// total ########--  80%  1/2 tarefas
}
