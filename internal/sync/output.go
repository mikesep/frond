package sync

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type actionEventType string

const (
	actionFailed    actionEventType = "FAIL"
	actionIgnored   actionEventType = "ign "
	actionSucceeded actionEventType = "ok  "
)

type actionEvent struct {
	Type    actionEventType
	Name    string
	Message string
}

type reporter interface {
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
	}

	r.printProgressLine()
	fmt.Fprintf(r.output, "\n") // leave space for event line

	return r
}

type ansiReporter struct {
	output   io.Writer
	total    int
	countLen int
	nameLen  int

	done      int
	failed    []actionEvent
	ignored   []actionEvent
	succeeded int
}

func (r *ansiReporter) HandleEvent(event actionEvent) {
	r.done++

	fmt.Fprintf(r.output, "\x1b[2F") // up two lines
	r.printProgressLine()

	switch event.Type {
	case actionFailed:
		r.failed = append(r.failed, event)
	case actionIgnored:
		r.ignored = append(r.ignored, event)
	case actionSucceeded:
		r.succeeded++
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

	fmt.Fprintf(r.output, "Done! %d failed, %d ignored, %d succeeded, %d total\n",
		len(r.failed), len(r.ignored), r.succeeded, r.total)

	for _, e := range append(r.failed, r.ignored...) {
		fmt.Fprintf(r.output, "  %s %-*s %s\n", e.Type, r.nameLen, e.Name, e.Message)
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

	done      int
	failed    int
	succeeded int
	ignored   int
}

func (r *plainReporter) HandleEvent(event actionEvent) {
	r.done++

	switch event.Type {
	case actionFailed:
		r.failed++
	case actionIgnored:
		r.ignored++
	case actionSucceeded:
		r.succeeded++
	default:
		panic(fmt.Sprintf("unexpected action event: %#v", event))
	}

	fmt.Fprintf(r.output,
		"[%*d/%d] %s %-*s %s\n",
		r.countLen, r.done, r.total,
		event.Type, r.nameLen, event.Name, event.Message,
	)
}

func (r *plainReporter) Done(note string) {
	if note != "" {
		fmt.Fprintf(r.output, "%s\n", note)
	}

	fmt.Fprintf(r.output,
		"Done! %d failed, %d ignored, %d succeeded, %d total\n",
		r.failed, r.ignored, r.succeeded, r.total)
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
