// Command ghwrapperrepro reproduces kunchenguid/no-mistakes#630 end to end on
// Linux: it puts a wrapper named `gh` on PATH that prints an audit notice to
// stderr on every invocation and then exits 0, exactly the shape of the
// reporter's Automic Vault build, and drives the real
// internal/scm/github.Host against it.
//
// Nothing here is mocked: every call spawns a process, which is the boundary
// the bug lives on.
//
// Exit code 0 means every gh payload survived the notice (fixed tree).
// Exit code 1 means at least one did not (bug present).
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kunchenguid/no-mistakes/internal/scm"
	"github.com/kunchenguid/no-mistakes/internal/scm/github"
)

const notice = "automic vault: human approval required"

// wrapper writes a `gh` that emits the notice on stderr and then answers like
// the real CLI would, exiting 0.
func wrapper(dir string) error {
	script := `#!/bin/sh
echo '` + notice + `' >&2
cat >/dev/null 2>&1 || true
case "$1 $2" in
"pr list")
	printf '%s\n' '[{"number":42,"url":"https://github.com/test/repo/pull/42","baseRefName":"main"}]'
	;;
"pr create")
	printf '%s\n' 'https://github.com/test/repo/pull/43'
	;;
"pr checks")
	printf '%s\n' '[{"name":"build","state":"SUCCESS","bucket":"pass"}]'
	;;
*)
	echo "unexpected gh invocation: $*" >&2
	exit 1
	;;
esac
exit 0
`
	path := filepath.Join(dir, "gh")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		return err
	}
	return nil
}

func main() {
	dir, err := os.MkdirTemp("", "gh-wrapper-")
	if err != nil {
		fmt.Println("FAIL: temp dir:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	if err := wrapper(dir); err != nil {
		fmt.Println("FAIL: write wrapper:", err)
		os.Exit(1)
	}
	// PATH is process-wide, so this must not race another goroutine here.
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
		fmt.Println("FAIL: set PATH:", err)
		os.Exit(1)
	}

	factory := func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, name, args...)
	}
	host := github.New(factory, nil, "", "test/repo")
	ctx := context.Background()
	failed := 0

	// 1. FindPR: the branch already has a PR, so the run must find it instead of
	// treating the payload as unparseable.
	pr, err := host.FindPR(ctx, "feature/wrapper", "main")
	switch {
	case err != nil:
		fmt.Printf("FAIL FindPR: %v\n", err)
		failed++
	case pr == nil:
		fmt.Println("FAIL FindPR: no PR found, so the pipeline would open a duplicate")
		failed++
	case pr.Number != "42" || pr.URL != "https://github.com/test/repo/pull/42":
		fmt.Printf("FAIL FindPR: got %+v, want #42 with a clean URL\n", pr)
		failed++
	default:
		fmt.Printf("ok   FindPR: #%s %s\n", pr.Number, pr.URL)
	}

	// 2. CreatePR: the created URL is the value, so a notice in it breaks every
	// later `gh pr view <url>`.
	created, err := host.CreatePR(ctx, "feature/wrapper", "main", scm.PRContent{
		Title: "fix: url",
		Body:  "## What Changed\n\n- keep the URL out of wrapper noise",
	})
	switch {
	case err != nil:
		fmt.Printf("FAIL CreatePR: %v\n", err)
		failed++
	case created.URL != "https://github.com/test/repo/pull/43" || created.Number != "43":
		fmt.Printf("FAIL CreatePR: URL = %q (number %q), want a clean #43\n", created.URL, created.Number)
		failed++
	default:
		fmt.Printf("ok   CreatePR: #%s %s\n", created.Number, created.URL)
	}

	// 3. GetChecks: the CI step polls this in a loop, so one parse failure per
	// iteration means the poll can never converge on what CI reported.
	for attempt := 1; attempt <= 3; attempt++ {
		checks, err := host.GetChecks(ctx, &scm.PR{Number: "42"})
		switch {
		case err != nil:
			fmt.Printf("FAIL GetChecks poll %d: %v\n", attempt, err)
			failed++
		case len(checks) != 1 || checks[0].Name != "build":
			fmt.Printf("FAIL GetChecks poll %d: checks = %+v, want the build check\n", attempt, checks)
			failed++
		default:
			fmt.Printf("ok   GetChecks poll %d: %s (%s)\n", attempt, checks[0].Name, checks[0].Bucket)
		}
	}

	fmt.Println(strings.Repeat("-", 60))
	if failed > 0 {
		fmt.Printf("REPRODUCED: %d gh payload(s) corrupted by a wrapper's stderr notice (issue #630)\n", failed)
		os.Exit(1)
	}
	fmt.Println("NOT REPRODUCED: every gh payload survived the wrapper's stderr notice")
}