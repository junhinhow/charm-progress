# charm-progress

A multi-task progress dashboard for Charmbracelet-style CLIs: one bar per job, plus a weighted total and ETA.

`bubbles/progress` renders a single bar. `charm-progress` keeps an ordered set of tasks (downloads, conversions, uploads) and renders the whole batch in a fixed width.

## Install

```bash
go get github.com/junhinhow/charm-progress
```

Requires Go 1.25 or newer.

## Usage

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
		if err := download(d, url, filepath.Join("downloads", filepath.Base(url))); err != nil {
			log.Printf("%s: %v", url, err)
		}
		// Re-render whenever you want: on a ticker, after a chunk, or from
		// Bubble Tea's Update. The dashboard is pure state + rendering.
		fmt.Println(d.View(80))
	}

	if d.Finished() {
		fmt.Println("all downloads finished")
	}
}

func download(d *progress.Dashboard, url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		d.Fail(url, err)
		return err
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		d.Fail(url, err)
		return err
	}
	file, err := os.Create(dest)
	if err != nil {
		d.Fail(url, err)
		return err
	}
	defer file.Close()

	// ContentLength is -1 when the server does not report it. Adding the same
	// ID again replaces the pending task in place and keeps insertion order.
	d.Add(progress.Task{
		ID:    url,
		Label: filepath.Base(dest),
		Total: resp.ContentLength,
		State: progress.Running,
	})

	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := file.Write(buf[:n]); err != nil {
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

Rendered output (ASCII theme, width 46):

```text
> baixar-1            ######----  60/100   60%
+ baixar-2            ########## 100/100  100%
total ########--  80%  1/2 tarefas
```

## API

| Symbol | Purpose |
| --- | --- |
| `State`, `Pending`, `Running`, `Done`, `Failed`, `Cancelled` | Task lifecycle. |
| `Task` | ID, label, total/current units, state and last error. |
| `Task.Percent()` | Clamped 0–100 progress; `Total <= 0` yields 0. |
| `Task.Remaining()` | Non-negative units left. |
| `New(opts...)` | Creates a dashboard (default theme, 20-column bar). |
| `Dashboard.Add` | Appends a task; a duplicate ID replaces it in place. |
| `Dashboard.Update` / `Advance` | Sets or adds progress; the first call starts the ETA clock. |
| `Dashboard.Done` / `Fail` / `Cancel` | Terminal transitions. |
| `Dashboard.Remove` | Removes a task and reports whether it existed. |
| `Dashboard.Task` / `Tasks` | Snapshot lookup and ordered copy. |
| `Dashboard.Overall` | Weighted 0–100 aggregate. |
| `Dashboard.ETA(now)` | Remaining time for running tasks; `0` when unknown. |
| `Dashboard.Active` / `Failed` / `Finished` | Counters and completion check. |
| `Dashboard.View(width)` | Full dashboard, one line per task plus a total line. |
| `Dashboard.ViewTask(t, width)` | Renders a single task line. |
| `Theme`, `DefaultTheme`, `ASCIiTheme` | Unicode/color theme and plain-ASCII fallback. |
| `WithTheme`, `WithBarWidth`, `WithShowETA`, `WithShowCounts`, `WithMaxVisible` | Options. |

## Design notes

- **Deterministic order.** Tasks live in a slice; the ID map is used only for lookup. Rendering never iterates a map.
- **Aggregate progress.** `Overall` is `sum(current) / sum(total)` over tasks with `Total > 0`, clamped to 0–100. If every total is zero, it is the share of `Done` tasks. Failed and cancelled tasks contribute the progress they reached, not 100%.
- **ETA.** The first `Update` records `startedAt`. `ETA(now)` uses the progress of `Running` tasks with a positive total and the elapsed time since that mark, assuming a constant rate. It returns `0` when there is no running task, no elapsed time, or no measured progress. Pass your own clock to keep tests deterministic; `View` uses `time.Now`.
- **Width discipline.** Widths are measured with `lipgloss.Width` and lines are cut with `ansi.Truncate`; `View(0)` and negative widths return `""`. On narrow terminals the layout first drops numeric counts that would hide the bar, then the bar itself, then falls back to symbol + percentage. No line ever exceeds the requested width.
- **Labels.** Control characters (including newlines and ANSI escapes) become spaces, so one task always occupies one line. Unicode labels are preserved as-is.
- **Fixed summary text.** The summary words are currently Portuguese (`total`, `tarefas`, `ETA`); only task labels are caller-provided. This keeps the package dependency-free and matches the rest of the suite.
- **Single goroutine.** The dashboard has no locks and no background goroutines. Drive it from one goroutine (or guard it yourself) and re-render when convenient.
- **ASCII fallback.** `ASCIiTheme` uses `#`/`-` and `. > + x -` for states, with no non-ASCII glyphs. It does not transliterate user labels.
- **No I/O.** The package never writes to stdout, never starts timers and never downloads anything; it only holds state and renders strings. The name's odd capitalization (`ASCIiTheme`) is part of the public API.

## License

MIT
