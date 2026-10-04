# PR: read `gh` stdout alone so a wrapper's stderr notice stops corrupting payloads

Fixes #630.

Credit for the report and the fix shape: **@alexlopespereira**, who filed
[#630](https://github.com/kunchenguid/no-mistakes/issues/630) with the exact
call sites, the three concrete failure symptoms, and the `ghOutput` helper to
use. This PR implements that shape and extends the same one-line rule to the
remaining data-bearing `gh` reads in the same provider.

## The bug

Every data-bearing `gh` invocation in the GitHub provider read
`CombinedOutput()` and then treated the merged bytes as data. `gh` is not
necessarily the process that runs: a wrapper around it — an approval gate, an
SSO helper, a corporate proxy — may print its own notice to stderr and still
exit 0, because the notice is not a refusal.

```
automic vault: human approval required     # stderr, exit 0
[{"number":42,"url":"https://github.com/org/repo/pull/42",...}]   # stdout
```

Merged, that payload no longer parses:

| call | command | symptom |
| --- | --- | --- |
| `FindPR` | `gh pr list --json …` | `parse gh pr list JSON: invalid character 'a'` — a branch's existing PR stops being found |
| `CreatePR` | `gh pr create` | the notice lands inside `pr.URL`, so every later `gh pr view <url>` targets a URL that never existed |
| `getPRChecks` | `gh pr checks --json …` | `parse CI checks: invalid character 'a'` — the CI step's poll loop cannot converge on any iteration |
| `getCommitChecks` | `gh api graphql` (rollup) | `parse checks for head commit: …` |
| `getWorkflowRunChecks` | `gh api … --paginate` | `parse workflow runs …` |
| `GetReviewComments` | `gh api graphql` | `decode PR review comments JSON: …` |
| `userAssetRepo` | `gh api graphql` | `parse repository for user-attachments: …` |

The last four are the same defect reached through other entry points, so the fix
covers them too rather than leaving a package that is half merged and half
isolated.

`UpdatePR`, `SetPRBaseBranch` and `RerunCheck` deliberately keep
`CombinedOutput()`: they only ever use the bytes to build an error message, and
merging is correct when the bytes are never parsed.

## The fix

One helper in `internal/scm/github/github.go` (doc comment and one inline note
abridged):

```go
func ghOutput(cmd *exec.Cmd) ([]byte, string, error) {
	out, err := cmd.Output()
	detail := strings.TrimSpace(string(out))
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if stderr := strings.TrimSpace(string(exitErr.Stderr)); stderr != "" {
			detail = stderr
		}
	}
	return out, detail, err
}
```

stdout alone is the payload; `detail` is what a failure quotes. Two consequences
worth calling out:

- **The `"no checks reported"` branch keeps working.** `gh pr checks` reports an
  empty rollup as a non-zero exit with that message on **stderr**, and the old
  code only found it because the streams were merged. `getPRChecks` now matches
  against `detail`, which is stderr for a failed call, so the branch is intact.
- **Diagnostics do not get worse.** A failing `gh` still surfaces its own stderr
  in the error, and `TestGetChecksSurfacesGHErrorStderr` (added for #644)
  keeps passing unchanged.

The rationale lives in the `ghOutput` doc comment, which is where
`AGENTS.md` says provider CLI traps are owned.

## Reproduction

Real processes, no mocks: a `gh` wrapper on `PATH` that writes the notice to
stderr and exits 0, driving the real `internal/scm/github.Host`.

Before (`f4b85ae`):

```
FAIL FindPR: parse gh pr list JSON: invalid character 'a' looking for beginning of value
FAIL CreatePR: URL = "automic vault: human approval required\nhttps://github.com/test/repo/pull/43" (number "43")
FAIL GetChecks poll 1: parse CI checks: invalid character 'a' looking for beginning of value
FAIL GetChecks poll 2: parse CI checks: invalid character 'a' looking for beginning of value
FAIL GetChecks poll 3: parse CI checks: invalid character 'a' looking for beginning of value
REPRODUCED: 5 gh payload(s) corrupted by a wrapper's stderr notice (issue #630)
```

After:

```
ok   FindPR: #42 https://github.com/test/repo/pull/42
ok   CreatePR: #43 https://github.com/test/repo/pull/43
ok   GetChecks poll 1: build (pass)
ok   GetChecks poll 2: build (pass)
ok   GetChecks poll 3: build (pass)
NOT REPRODUCED: every gh payload survived the wrapper's stderr notice
```

## Tests

New regression tests; the first six fail on `f4b85ae` without the change
(verified by stashing only the two source files and re-running):

- `TestFindPRIgnoresWrapperStderrNotice`
- `TestCreatePRKeepsURLOutOfWrapperStderrNotice`
- `TestGetChecksIgnoresWrapperStderrNotice`
- `TestGetChecksReadsCommitRollupAndRunsDespiteWrapperStderrNotice`
- `TestGetReviewCommentsIgnoresWrapperStderrNotice`
- `TestUserAssetRepoIgnoresWrapperStderrNotice`
- `TestGetChecksKeepsNoChecksReportedOnStderr` — pins the branch that only
  worked because the streams were merged; passes before and after, and exists
  so a later "simplification" cannot drop it.

They use the existing `githubTestCmdFactory` / `githubTestResponse` fixture,
which already supports `stderr` with `code: 0`, so no new test machinery was
needed.

## Verification

```
gofmt -l .                                    # clean
make lint                                     # genskill --check + go vet, clean
go test ./internal/scm/github/                # ok
go test -race ./...                           # ok, 52 packages, 4m06s
go build -o ./bin/no-mistakes ./cmd/no-mistakes
```

`make e2e` was not run: this change touches no agent integration, no e2e
harness and no recorded fixture, and it is behind the `e2e` build tag.

---

**Do not submit this body as-is.** `CONTRIBUTING.md` requires every PR to be
raised through `no-mistakes` itself, and the `Require no-mistakes` workflow
fails a body that lacks the deterministic signature plus the parseable v1
pipeline attestation. So the real submission has to be
`no-mistakes init --fork-url …` → `git push no-mistakes`, with no-mistakes
writing the `## Pipeline` section over this text. That needs a configured,
authenticated agent CLI, so it is left to a human with that set up. Keep the
sections above the `## Pipeline` section when that happens; replacing the whole
body removes the signature and the check fails until no-mistakes rewrites it.

**Before pushing:** the patch was generated from a local commit whose author is a
placeholder (`no-mistakes contributor <noreply@example.invalid>`), because this
work was done without a GitHub identity. Re-author it with your own
(`git commit --amend --reset-author --author "Your Name <you@example.com>"`) so the
commit is attributed to you and not to a placeholder.