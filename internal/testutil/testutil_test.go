package testutil_test

import (
	"testing"
	"time"

	"github.com/invidian/golang-cli-testing-example/internal/testutil"
)

// Tests for test helper are a little bit extreme, but they serve the purpose here to
// avoid having duplicated helpers in the code and being able to reach 100% mutation score
// on the code at the same time.
func Test_ContextWithDeadline(t *testing.T) {
	t.Parallel()

	testT := &testTesting{
		time: time.Now(),
	}

	ctx := testutil.ContextWithDeadline(testT)

	t.Run("calls_helper_method", func(t *testing.T) {
		t.Parallel()

		if !testT.helper {
			t.Fatalf("Expected helper call")
		}
	})

	t.Run("adds_cancel_function_cleanup", func(t *testing.T) {
		t.Parallel()

		if testT.cleanup == nil {
			t.Fatalf("Expected cleanup function to be registered")
		}
	})

	t.Run("returns_context_with_deadline_when_test_has_deadline_set", func(t *testing.T) {
		t.Parallel()

		if _, ok := ctx.Deadline(); !ok {
			t.Fatalf("Received context has no deadline set")
		}
	})
}

type testTesting struct {
	helper  bool
	cleanup func()
	time    time.Time
}

func (t *testTesting) Helper() {
	t.helper = true
}

func (t *testTesting) Cleanup(f func()) {
	t.cleanup = f
}

func (t *testTesting) Deadline() (time.Time, bool) {
	return t.time, !t.time.IsZero()
}
