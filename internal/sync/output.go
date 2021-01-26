package sync

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type actionEventType string

const (
	actionCloned    actionEventType = "new "
	actionFailed    actionEventType = "FAIL"
	actionIgnored   actionEventType = "ign "
	actionRemoved   actionEventType = "rm  "
	actionUnchanged actionEventType = "ok  "
	actionUpdated   actionEventType = "upd "
)

type actionEvent struct {
	Type    actionEventType
	Name    string
	Message string
	Caveats []string
}

type reporter interface {
	DrawInitial()
	HandleEvent(actionEvent)
	Done(note string)
	NumFailed() int
}

//------------------------------------------------------------------------------

type serializingReporter struct {
	q    chan<- actionEvent
	done <-chan struct{}

	next reporter
}

func newSerializingReporter(next reporter) *serializingReporter {
	q := make(chan actionEvent)
	done := make(chan struct{})

	go func() {
		for e := range q {
			next.HandleEvent(e)
		}
		close(done)
	}()

	return &serializingReporter{q: q, done: done, next: next}
}

func (r *serializingReporter) DrawInitial() {
	r.next.DrawInitial()
}

func (r *serializingReporter) HandleEvent(event actionEvent) {
	r.q <- event
}

func (r *serializingReporter) Done(note string) {
	close(r.q)
	<-r.done
	r.next.Done(note)
}

func (r *serializingReporter) NumFailed() int {
	return r.next.NumFailed()
}

//------------------------------------------------------------------------------

func newANSIReporter(w io.Writer, totalItems int, maxNameLen int) *ansiReporter {
	r := &ansiReporter{
		output:   w,
		total:    totalItems,
		countLen: len(strconv.Itoa(totalItems)),
		nameLen:  maxNameLen,
		failed:   make([]actionEvent, 0, totalItems),
		ignored:  make([]actionEvent, 0, totalItems),
		caveats:  make(map[string][]string),
	}

	return r
}

type ansiReporter struct {
	output   io.Writer
	total    int
	countLen int
	nameLen  int

	done int

	cloned    int
	failed    []actionEvent
	ignored   []actionEvent
	removed   int
	unchanged int
	updated   int

	caveats map[string][]string // repo name -> list of caveats
}

func (r *ansiReporter) DrawInitial() {
	r.printProgressLine()
	fmt.Fprintf(r.output, "\n") // leave space for event line
}

func (r *ansiReporter) HandleEvent(event actionEvent) {
	r.done++

	fmt.Fprintf(r.output, "\x1b[2F") // up two lines
	r.printProgressLine()

	switch event.Type {
	case actionCloned:
		r.cloned++
	case actionFailed:
		r.failed = append(r.failed, event)
	case actionIgnored:
		r.ignored = append(r.ignored, event)
	case actionRemoved:
		r.removed++
	case actionUnchanged:
		r.unchanged++
	case actionUpdated:
		r.updated++
	default:
		panic(fmt.Sprintf("unexpected event: %#v", event))
	}

	if event.Caveats != nil {
		r.caveats[event.Name] = event.Caveats
	}

	fmt.Fprintf(r.output, "%s %-*s %s", event.Type, r.nameLen, event.Name, event.Message)
	fmt.Fprintf(r.output, "\x1b[0K\n") // from cursor until the end of the line, then \n
}

func (r *ansiReporter) Done(note string) {
	fmt.Fprintf(r.output, "\x1b[1F") // up one line to overwrite the last repo event
	fmt.Fprintf(r.output, "\x1b[0K") // clear the line

	if note != "" {
		fmt.Fprintf(r.output, "%s\n", note)
	}

	fmt.Fprintf(r.output, "Done! ")
	if r.cloned > 0 {
		fmt.Fprintf(r.output, "%d cloned, ", r.cloned)
	}
	if len(r.failed) > 0 {
		fmt.Fprintf(r.output, "%d FAILED, ", len(r.failed))
	}
	if len(r.ignored) > 0 {
		fmt.Fprintf(r.output, "%d ignored, ", len(r.ignored))
	}
	if r.removed > 0 {
		fmt.Fprintf(r.output, "%d removed, ", r.removed)
	}
	if r.unchanged > 0 {
		fmt.Fprintf(r.output, "%d unchanged, ", r.unchanged)
	}
	if r.updated > 0 {
		fmt.Fprintf(r.output, "%d updated, ", r.updated)
	}
	fmt.Fprintf(r.output, "%d total\n", r.total)

	for _, e := range append(r.failed, r.ignored...) {
		fmt.Fprintf(r.output, "  %s %-*s %s\n", e.Type, r.nameLen, e.Name, e.Message)
	}

	if len(r.caveats) > 0 {
		fmt.Fprintf(r.output, "Caveats:\n")
		for repo, cavs := range r.caveats {
			for _, c := range cavs {
				fmt.Fprintf(r.output, "  %s: %s\n", repo, c)
			}
		}
	}
}

func (r *ansiReporter) NumFailed() int {
	return len(r.failed)
}

func (r *ansiReporter) printProgressLine() {
	const barLen = 60

	fmt.Fprintf(r.output,
		"%*d/%d [%-*s]",
		r.countLen, r.done, r.total,
		barLen,
		strings.Repeat("=", int(barLen*r.done/r.total)),
	)

	fmt.Fprintf(r.output, "\x1b[0K\n") // clear from cursor to the end of the line, then \n
}

//------------------------------------------------------------------------------

func newPlainReporter(w io.Writer, totalItems int, maxNameLen int) *plainReporter {
	return &plainReporter{
		output:   w,
		total:    totalItems,
		countLen: len(strconv.Itoa(totalItems)),
		nameLen:  maxNameLen,
	}
}

type plainReporter struct {
	output   io.Writer
	total    int
	countLen int
	nameLen  int

	cloned    int
	failed    int
	ignored   int
	removed   int
	unchanged int
	updated   int

	caveats int
	done    int
}

func (r *plainReporter) DrawInitial() {
	// nothing required
}

func (r *plainReporter) HandleEvent(event actionEvent) {
	r.done++

	switch event.Type {
	case actionCloned:
		r.cloned++
	case actionFailed:
		r.failed++
	case actionIgnored:
		r.ignored++
	case actionRemoved:
		r.removed++
	case actionUnchanged:
		r.unchanged++
	case actionUpdated:
		r.updated++
	default:
		panic(fmt.Sprintf("unexpected action event: %#v", event))
	}

	fmt.Fprintf(r.output,
		"[%*d/%d] %s %-*s %s\n",
		r.countLen, r.done, r.total,
		event.Type, r.nameLen, event.Name, event.Message,
	)

	r.caveats += len(event.Caveats)
	for _, caveat := range event.Caveats {
		fmt.Fprintf(r.output, "  %s\n", caveat)
	}
}

func (r *plainReporter) Done(note string) {
	if note != "" {
		fmt.Fprintf(r.output, "%s\n", note)
	}

	fmt.Fprintf(r.output, "Done! ")
	if r.cloned > 0 {
		fmt.Fprintf(r.output, "%d cloned, ", r.cloned)
	}
	if r.failed > 0 {
		fmt.Fprintf(r.output, "%d FAILED, ", r.failed)
	}
	if r.ignored > 0 {
		fmt.Fprintf(r.output, "%d ignored, ", r.ignored)
	}
	if r.removed > 0 {
		fmt.Fprintf(r.output, "%d removed, ", r.removed)
	}
	if r.unchanged > 0 {
		fmt.Fprintf(r.output, "%d unchanged, ", r.unchanged)
	}
	if r.updated > 0 {
		fmt.Fprintf(r.output, "%d updated, ", r.updated)
	}
	fmt.Fprintf(r.output, "%d total\n", r.total)

	if r.caveats > 0 {
		word := "caveats"
		if r.caveats == 1 {
			word = "caveat"
		}
		fmt.Fprintf(r.output, "See %d %s above.\n", r.caveats, word)
	}
}

func (r *plainReporter) NumFailed() int {
	return r.failed
}

//------------------------------------------------------------------------------

// type fancyOutputter struct {
// 	q chan []byte
// 	w io.Writer
// }

// func (fo *fancyOutputter) Write(p []byte) (n int, err error) {
// 	fo.queue <- p
// 	return len(p), nil
// }

// func (fo *fancyOutputter) Close() error {
// 	return fo.queue.Close()
// }

// func (fo *fancyOutputter) Start() {
// 	go func() {
// 		select {
// 		case msg <- fo.queue:
// 			fo.writer.Write(msg)
// 		default:
// 			fmt.Fprintf(fo.writer, "SOMETHING ELSE!\n")
// 		}
// 	}()
// }
