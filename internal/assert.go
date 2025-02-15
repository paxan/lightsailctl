package internal

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Assert(t testing.TB, what string, want, got any) {
	t.Helper()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("%s mismatch (-want +got):\n%s", what, diff)
	}
}

func AssertErr(t testing.TB, want string, got error) {
	t.Helper()

	gotStr := ""
	if got != nil {
		gotStr = got.Error()
	}

	Assert(t, "error", want, gotStr)
}
