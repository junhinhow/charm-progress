# charm-progress

Dashboard de progresso multi-tarefa para CLIs no estilo Charmbracelet: uma barra por tarefa, mais um total ponderado e ETA.

O `bubbles/progress` renderiza uma única barra. O `charm-progress` mantém um conjunto ordenado de tarefas (downloads, conversões, uploads) e renderiza o lote inteiro em uma largura fixa.

## Instalação

```bash
go get github.com/junhinhow/charm-progress
```

Requer Go 1.25 ou superior.

## Uso

```go
package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/junhinhow/charm-progress"
)

func main() {
	urls := []string{
		"https://example.com/files/a.zip",
		"https://example.com/files/b.zip",
	}

	d := progress.New(progress.WithBarWidth(24))
	for _, url := range urls {
		d.Add(progress.Task{
			ID:    url,
			Label: filepath.Base(url),
			State: progress.Pending,
		})
	}

	for _, url := range urls {
		if err := baixar(d, url, filepath.Join("downloads", filepath.Base(url))); err != nil {
			log.Printf("%s: %v", url, err)
		}
		// Renderize quando quiser: em um ticker, após um chunk ou no Update
		// do Bubble Tea. O dashboard é só estado + renderização.
		fmt.Println(d.View(80))
	}

	if d.Finished() {
		fmt.Println("todos os downloads terminaram")
	}
}

func baixar(d *progress.Dashboard, url, destino string) error {
	resp, err := http.Get(url)
	if err != nil {
		d.Fail(url, err)
		return err
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
		d.Fail(url, err)
		return err
	}
	arquivo, err := os.Create(destino)
	if err != nil {
		d.Fail(url, err)
		return err
	}
	defer arquivo.Close()

	// ContentLength é -1 quando o servidor não informa o tamanho. Adicionar o
	// mesmo ID novamente substitui a tarefa pendente no lugar e preserva a
	// ordem de inserção.
	d.Add(progress.Task{
		ID:    url,
		Label: filepath.Base(destino),
		Total: resp.ContentLength,
		State: progress.Running,
	})

	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := arquivo.Write(buf[:n]); err != nil {
				d.Fail(url, err)
				return err
			}
			d.Advance(url, int64(n))
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			d.Fail(url, readErr)
			return readErr
		}
	}

	d.Done(url)
	return nil
}
```

Saída renderizada (tema ASCII, largura 46):

```text
> baixar-1            ######----  60/100   60%
+ baixar-2            ########## 100/100  100%
total ########--  80%  1/2 tarefas
```

## API

| Símbolo | Finalidade |
| --- | --- |
| `State`, `Pending`, `Running`, `Done`, `Failed`, `Cancelled` | Ciclo de vida da tarefa. |
| `Task` | ID, rótulo, unidades atual/total, estado e último erro. |
| `Task.Percent()` | Progresso limitado a 0–100; `Total <= 0` devolve 0. |
| `Task.Remaining()` | Unidades restantes, nunca negativas. |
| `New(opts...)` | Cria um dashboard (tema padrão, barra de 20 colunas). |
| `Dashboard.Add` | Acrescenta uma tarefa; ID duplicado substitui no mesmo lugar. |
| `Dashboard.Update` / `Advance` | Define ou soma progresso; a primeira chamada inicia o relógio do ETA. |
| `Dashboard.Done` / `Fail` / `Cancel` | Transições finais. |
| `Dashboard.Remove` | Remove uma tarefa e informa se ela existia. |
| `Dashboard.Task` / `Tasks` | Consulta pontual e cópia ordenada. |
| `Dashboard.Overall` | Agregado ponderado de 0–100. |
| `Dashboard.ETA(now)` | Tempo restante das tarefas em execução; `0` quando desconhecido. |
| `Dashboard.Active` / `Failed` / `Finished` | Contadores e verificação de conclusão. |
| `Dashboard.View(width)` | Dashboard completo, uma linha por tarefa mais o total. |
| `Dashboard.ViewTask(t, width)` | Renderiza uma única linha de tarefa. |
| `Theme`, `DefaultTheme`, `ASCIiTheme` | Tema Unicode/colorido e alternativa ASCII pura. |
| `WithTheme`, `WithBarWidth`, `WithShowETA`, `WithShowCounts`, `WithMaxVisible` | Opções. |

## Notas de design

- **Ordem determinística.** As tarefas ficam em um slice; o mapa de IDs serve apenas para busca. A renderização nunca itera um mapa.
- **Progresso agregado.** `Overall` é `soma(current) / soma(total)` sobre tarefas com `Total > 0`, limitado a 0–100. Se todos os totais forem zero, é a fração de tarefas `Done`. Tarefas com falha ou canceladas contribuem com o progresso alcançado, não com 100%.
- **ETA.** O primeiro `Update` registra `startedAt`. `ETA(now)` usa o progresso das tarefas `Running` com total positivo e o tempo decorrido desde essa marca. Devolve `0` quando não há tarefa em execução, tempo decorrido ou progresso medido. Passe seu próprio relógio para testes determinísticos; `View` usa `time.Now`.
- **Disciplina de largura.** As larguras são medidas com `lipgloss.Width` e as linhas são cortadas com `ansi.Truncate`; `View(0)` e larguras negativas devolvem `""`. Em terminais estreitos o layout primeiro remove as contagens numéricas que esconderiam a barra, depois a própria barra e, por fim, recorre a símbolo + percentual. Nenhuma linha excede a largura pedida.
- **Rótulos.** Caracteres de controle (incluindo quebras de linha e escapes ANSI) viram espaços, então cada tarefa ocupa sempre uma linha. Rótulos Unicode são preservados como estão.
- **Uma goroutine.** O dashboard não tem locks nem goroutines próprias. Use-o de uma única goroutine (ou proteja o acesso) e re-renderize quando fizer sentido.
- **Alternativa ASCII.** `ASCIiTheme` usa `#`/`-` e `. > + x -` para os estados, sem glifos não-ASCII. Ele não translitera rótulos do usuário.
- **Sem I/O.** O pacote nunca escreve em stdout, nunca inicia timers e nunca baixa nada; apenas guarda estado e renderiza strings. A grafia incomum do nome (`ASCIiTheme`) faz parte da API pública.

## Licença

MIT
