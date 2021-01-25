package sync

import (
	"os"
	"testing"
	"time"
)

func TestPlainOutputter(t *testing.T) {
	r := newPlainReporter(os.Stderr, 4, 5)

	r.HandleEvent(actionEvent{Type: actionSucceeded, Name: "alice"})
	r.HandleEvent(actionEvent{Type: actionSucceeded, Name: "bob"})
	r.HandleEvent(actionEvent{Type: actionIgnored, Name: "skip", Message: "it's in the name"})
	r.HandleEvent(actionEvent{Type: actionFailed, Name: "crash", Message: "oh no!"})

	r.Done("")
}

func TestSerializedPlainReporter(t *testing.T) {
	plain := newPlainReporter(os.Stderr, 4, 5)

	r := newSerializingReporter(plain)

	r.HandleEvent(actionEvent{Type: actionSucceeded, Name: "alice"})
	r.HandleEvent(actionEvent{Type: actionSucceeded, Name: "bob"})
	r.HandleEvent(actionEvent{Type: actionIgnored, Name: "skip", Message: "it's in the name"})
	r.HandleEvent(actionEvent{Type: actionFailed, Name: "crash", Message: "oh no!"})

	r.Done("")
}

func TestSerializedANSIReporter(t *testing.T) {
	events := []actionEvent{
		{Type: actionSucceeded, Name: "arlington"},
		{Type: actionSucceeded, Name: "boston"},
		{Type: actionSucceeded, Name: "chicago"},
		{Type: actionSucceeded, Name: "dalles"},
		{Type: actionSucceeded, Name: "encino"},
		{Type: actionSucceeded, Name: "frankfort"},
		{Type: actionSucceeded, Name: "georgetown"},
		{Type: actionSucceeded, Name: "harrisburg"},
		{Type: actionSucceeded, Name: "indianapolis"},
		{Type: actionSucceeded, Name: "juneau"},
		{Type: actionSucceeded, Name: "kalamazoo"},
		{Type: actionSucceeded, Name: "louisville"},
		{Type: actionSucceeded, Name: "minneapolis"},
		{Type: actionIgnored, Name: "skip", Message: "it's in the name"},
		{Type: actionFailed, Name: "crash", Message: "oh no!"},
	}
	maxNameLen := 0
	for _, e := range events {
		if len(e.Name) > maxNameLen {
			maxNameLen = len(e.Name)
		}
	}

	ansi := newANSIReporter(os.Stderr, len(events), maxNameLen)

	r := newSerializingReporter(ansi)

	delay := 100 * time.Millisecond
	for _, e := range events {
		time.Sleep(delay)
		r.HandleEvent(e)
	}

	r.Done("")
}
