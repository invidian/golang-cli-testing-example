// Package testutil provides common test helpers which are not intended to be consumed outside of this module.
package testutil

import (
	"context"
	"time"
)

const (
	// Arbitrary amount of time to let tests exit cleanly before main process terminates.
	timeoutGracePeriod = 10 * time.Second
)

// Testing is an interface representing testing.T, so helpers itself can be tested as well.
type Testing interface {
	Helper()
	Deadline() (time.Time, bool)
	Cleanup(func())
}

// ContextWithDeadline returns context which will timeout before t.Deadline().
//
//nolint:varnamelen // Make exception for t, as it should be treated as *testing.T still.
// We use Testing instead of *t.testing.T to be able to test this code.
func ContextWithDeadline(t Testing) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	deadline, ok := t.Deadline()
	if ok {
		ctx, cancel = context.WithDeadline(context.Background(), deadline.Truncate(timeoutGracePeriod))
	}

	t.Cleanup(cancel)

	return ctx
}
