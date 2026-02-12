package util

import (
	"io"
	"os"
	"testing"
)

func captureStderr(t *testing.T) func() string {
	t.Helper()

	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	os.Stderr = w

	return func() string {
		t.Helper()
		_ = w.Close()
		os.Stderr = orig
		b, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}
		_ = r.Close()
		return string(b)
	}
}

func TestNewDebugDefaultAndNamespaceOutput(t *testing.T) {
	d := newDebugNotInitialized("test:ns")
	done := captureStderr(t)

	d.Print("before")
	applyDebug("golar:test:ns", []*Debug{d})
	d.Print("after")

	got := done()
	want := "golar:test:ns after\n"
	if got != want {
		t.Fatalf("unexpected stderr output:\nwant %q\ngot  %q", want, got)
	}
}

func TestApplyDebugMatchesAndSkips(t *testing.T) {
	matched := newDebugNotInitialized("api:users")
	skipped := newDebugNotInitialized("api:internal")
	unmatched := newDebugNotInitialized("worker:queue")
	done := captureStderr(t)

	applyDebug("golar:api:*,-golar:api:internal", []*Debug{matched, skipped, unmatched})
	matched.Print("matched")
	skipped.Print("skipped")
	unmatched.Print("unmatched")

	got := done()
	want := "golar:api:users matched\n"
	if got != want {
		t.Fatalf("unexpected stderr output:\nwant %q\ngot  %q", want, got)
	}
}

func TestApplyDebugLeadingSkipThenMatch(t *testing.T) {
	exact := newDebugNotInitialized("api")
	child := newDebugNotInitialized("api:users")
	done := captureStderr(t)

	applyDebug("-golar:api,golar:api*", []*Debug{exact, child})
	exact.Print("exact")
	child.Print("child")

	got := done()
	want := "golar:api:users child\n"
	if got != want {
		t.Fatalf("unexpected stderr output:\nwant %q\ngot  %q", want, got)
	}
}

func TestApplyDebugEmptyStringNoOp(t *testing.T) {
	api := newDebugNotInitialized("api")
	worker := newDebugNotInitialized("worker")
	done := captureStderr(t)

	applyDebug("golar:api", []*Debug{api, worker})
	applyDebug("", []*Debug{api, worker})
	api.Print("api")
	worker.Print("worker")

	got := done()
	want := "golar:api api\n"
	if got != want {
		t.Fatalf("unexpected stderr output:\nwant %q\ngot  %q", want, got)
	}
}

func TestApplyDebugSkipsEmptyParts(t *testing.T) {
	api := newDebugNotInitialized("api")
	worker := newDebugNotInitialized("worker")
	done := captureStderr(t)

	applyDebug(",,golar:api,,", []*Debug{api, worker})
	api.Print("api")
	worker.Print("worker")

	got := done()
	want := "golar:api api\n"
	if got != want {
		t.Fatalf("unexpected stderr output:\nwant %q\ngot  %q", want, got)
	}
}

func TestApplyDebugSkipsBareDashPart(t *testing.T) {
	api := newDebugNotInitialized("api")
	worker := newDebugNotInitialized("worker")
	done := captureStderr(t)

	applyDebug("-,golar:api", []*Debug{api, worker})
	api.Print("api")
	worker.Print("worker")

	got := done()
	want := "golar:api api\n"
	if got != want {
		t.Fatalf("unexpected stderr output:\nwant %q\ngot  %q", want, got)
	}
}

func TestApplyDebugAddsAlternationForMultipleParts(t *testing.T) {
	api := newDebugNotInitialized("api")
	worker := newDebugNotInitialized("worker")
	internal := newDebugNotInitialized("internal")
	other := newDebugNotInitialized("other")
	done := captureStderr(t)

	applyDebug("golar:api,golar:worker,-golar:internal,-golar:other", []*Debug{api, worker, internal, other})
	api.Print("api")
	worker.Print("worker")
	internal.Print("internal")
	other.Print("other")

	got := done()
	want := "golar:api api\ngolar:worker worker\n"
	if got != want {
		t.Fatalf("unexpected stderr output:\nwant %q\ngot  %q", want, got)
	}
}

func TestApplyDebugSkipOnlyReturnsWithoutChanges(t *testing.T) {
	api := newDebugNotInitialized("api")
	worker := newDebugNotInitialized("worker")
	done := captureStderr(t)

	applyDebug("golar:api", []*Debug{api, worker})
	applyDebug("-golar:api,-golar:worker", []*Debug{api, worker})
	api.Print("api")
	worker.Print("worker")

	got := done()
	want := "golar:api api\n"
	if got != want {
		t.Fatalf("unexpected stderr output:\nwant %q\ngot  %q", want, got)
	}
}

func TestApplyDebugWildcardMatchWithWildcardSkip(t *testing.T) {
	web := newDebugNotInitialized("service:web")
	dbRead := newDebugNotInitialized("service:db:read")
	dbWrite := newDebugNotInitialized("service:db:write")
	done := captureStderr(t)

	applyDebug("golar:service:*,-golar:service:db*", []*Debug{web, dbRead, dbWrite})
	web.Print("web")
	dbRead.Print("db-read")
	dbWrite.Print("db-write")

	got := done()
	want := "golar:service:web web\n"
	if got != want {
		t.Fatalf("unexpected stderr output:\nwant %q\ngot  %q", want, got)
	}
}

func TestPrintAndPrintf(t *testing.T) {
	done := captureStderr(t)
	d := newDebugNotInitialized("test:ns")

	applyDebug("golar:test:ns", []*Debug{d})
	d.Print("hello", "world")
	d.Printf("value=%d", 7)

	got := done()
	want := "golar:test:ns hello world\ngolar:test:ns value=7\n"
	if got != want {
		t.Fatalf("unexpected stderr output:\nwant %q\ngot  %q", want, got)
	}
}

func TestPrintAndPrintfWhenNotMatched(t *testing.T) {
	done := captureStderr(t)
	d := newDebugNotInitialized("test:disabled")

	applyDebug("golar:other", []*Debug{d})
	d.Print("hello")
	d.Printf("value=%d", 7)

	if got := done(); got != "" {
		t.Fatalf("expected no output for unmatched debug, got %q", got)
	}
}
