// SPDX-FileCopyrightText: 2021 Michael Seplowitz
// SPDX-License-Identifier: MIT

package sync

import (
	"os"
	"testing"
	"time"
)

func TestPlainOutputter(t *testing.T) {
	r := newPlainReporter(os.Stderr, 4, 5)

	r.HandleEvent(actionEvent{Type: actionUpdated, Name: "alice"})
	r.HandleEvent(actionEvent{Type: actionUnchanged, Name: "bob"})
	r.HandleEvent(actionEvent{Type: actionIgnored, Name: "skip", Message: "it's in the name"})
	r.HandleEvent(actionEvent{Type: actionFailed, Name: "crash", Message: "oh no!"})

	r.Done("")
}

func TestSerializedPlainReporter(t *testing.T) {
	plain := newPlainReporter(os.Stderr, 4, 5)

	r := newSerializingReporter(plain)

	r.HandleEvent(actionEvent{Type: actionUpdated, Name: "alice"})
	r.HandleEvent(actionEvent{Type: actionUnchanged, Name: "bob"})
	r.HandleEvent(actionEvent{Type: actionIgnored, Name: "skip", Message: "it's in the name"})
	r.HandleEvent(actionEvent{Type: actionFailed, Name: "crash", Message: "oh no!"})

	r.Done("")
}

func TestSerializedANSIReporter(t *testing.T) {
	events := []actionEvent{
		{Type: actionUpdated, Name: "arlington"},
		{Type: actionUpdated, Name: "boston"},
		{Type: actionUpdated, Name: "chicago"},
		{Type: actionUpdated, Name: "dalles"},
		{Type: actionUpdated, Name: "encino"},
		{Type: actionUpdated, Name: "frankfort"},
		{Type: actionUpdated, Name: "georgetown"},
		{Type: actionUpdated, Name: "harrisburg"},
		{Type: actionUpdated, Name: "indianapolis"},
		{Type: actionUpdated, Name: "juneau"},
		{Type: actionUpdated, Name: "kalamazoo"},
		{Type: actionUpdated, Name: "louisville"},
		{Type: actionUpdated, Name: "minneapolis"},
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
