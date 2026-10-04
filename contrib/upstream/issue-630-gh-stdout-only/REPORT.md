# REPORT — upstream contribution to `kunchenguid/no-mistakes`, pass 3

**Date of all retrievals: 2026-10-01 (UTC).** Sandbox egress IP `70.120.51.9`.
No credentials were used or requested; every source below is public.

---

## 1. What I produced

| artifact | path |
| --- | --- |
| Patch (plain `git diff` against upstream `main`) | `/work/patches/no-mistakes-630-gh-stdout-only.patch` |
| Patch (`git format-patch`, with commit message) | `/work/patches/0001-fix-scm-read-gh-stdout-only.patch` |
| PR description (credits the reporter) | `/work/PR_DESCRIPTION.md` |
| Pass-3 issue-selection record with the full 98-issue derivation | `/work/PASS3_SELECTION.md` |
| Working clone, on local branch `fix/gh-stdout-only-reads` (never pushed) | `/work/no-mistakes` |
| End-to-end reproduction harness | `/work/repro/run.sh`, `/work/repro/main.go` |
| Scraper scripts used to enumerate issues without the REST API | `/work/tools/scrape_issues.py`, `/work/tools/scrape_issue_detail.py` |
| Raw retrieved data | `/work/data/*.json` |

**Nothing was pushed.** No `git push`, no fork, no PR was opened; the branch and
commit exist only in the local clone at `/work/no-mistakes`. A human submits.

### The change

Base commit: `f4b85ae161cef8f14e289cc90aebae044ed570b9`
(`feat(config): add machine-local additional checks and nice priority to
repository_overrides (#1270)`), which was `origin/main` at both clone time and at
the final `git fetch` — verified with `git rev-list --left-right --count
origin/main...HEAD` = `0  1`.

```
internal/scm/github/attachments.go      |   4 +-
internal/scm/github/attachments_test.go |  21 ++++
internal/scm/github/github.go           |  57 ++++++++---
internal/scm/github/github_test.go      | 165 ++++++++++++++++++++++++++++++++
4 files changed, 232 insertions(+), 15 deletions(-)
```

Fixes issue **#630**, "gh wrappers that write to stderr corrupt FindPR, CreatePR
and GetChecks", filed by **@alexlopespereira**
(<https://github.com/kunchenguid/no-mistakes/issues/630>, retrieved
2026-10-01).

One helper, `ghOutput`, in `internal/scm/github/github.go`: `cmd.Output()`
returns stdout alone, stderr goes into the `detail` string that error messages
quote (via `exec.ExitError.Stderr`). It replaces `CombinedOutput()` at all seven
**data-bearing** `gh` reads in the GitHub provider — `FindPR`, `CreatePR`,
`getPRChecks`, `getCommitChecks`, `getWorkflowRunChecks`, `GetReviewComments`,
`userAssetRepo`. The three invocations that only use the bytes to build an error
message (`UpdatePR`, `SetPRBaseBranch`, `RerunCheck`) deliberately keep
`CombinedOutput()`.

Seven regression tests were added; six of them fail on the unfixed tree, and
`TestGetChecksKeepsNoChecksReportedOnStderr` pins the `"no checks reported"`
branch, which only ever worked because `gh` writes it to **stderr** with a
non-zero exit and the old code found it in the merged stream.

---

## 2. How I picked #630 (pass 3 = eligible issues 21–30)

Full derivation with per-issue reasons is in `/work/PASS3_SELECTION.md`. Summary:

- `is:issue state:open label:bug sort:created-asc` → **98** issues (GitHub's own
  count in the payload matched the 98 rows I scraped).
- Ineligible = carries the maintainer's `existing-pr` label, **or** the issue page
  shows an `OPEN` linked pull request. That excluded **11** issues, leaving **87
  eligible**. No open issue had an assignee or an agent assignment, so those
  signals excluded nothing.
- Eligible positions **21–30** are #602, #606, #610, #620, **#630**, #642, #658,
  #665, #667, #693.
- **#630 is the 25th oldest eligible** and the only one in that window that
  reproduces on Linux with no special hardware and no vendor credentials. Every
  other candidate needs something this sandbox does not have: an installed
  managed service (#610), an out-of-band history rewrite in a live gate worktree
  (#642, #693), two agent CLIs with different auth states on Windows (#658),
  Chrome to observe a rendering difference (#665), PowerShell-produced
  double-encoded log fixtures (#606), or a live pipeline run (#602, which also
  overlaps the open PR #1245 for #924). #620 already has a closed PR (#621) and
  its reporter says one is ready.

Re-checked the whole population 18 minutes after the first scrape: still 98
issues, zero label changes, zero new/removed issues, #630 still at eligible
position 25 (`/work/data/open-bug-issues-recheck.json`,
`fetched_at 2026-10-01T05:09:55Z`).

Fresh re-check of #630 itself at 2026-10-01T05:10:17Z
(`/work/data/issue-630-recheck.json`): `state: OPEN`, labels `bug` +
`ready-for-pr`, author `alexlopespereira`, `linked_pull_requests: []`,
`assigned: []`, `agent_assignments: []`, 5 timeline items, last activity
2026-09-19. All three maintainer triage stamps read `ready-for-pr`; the newest
says in its own words "**Still unfixed:** … No owning open PR." No remote branch
name contains `630` (`git ls-remote --heads origin`).

---

## 3. How I verified it

**Bug reproduces before the fix (real processes, no mocks).** `/work/repro/run.sh`
puts a `gh` wrapper on `PATH` that writes `automic vault: human approval required`
to stderr and exits 0 — the shape the reporter's Automic Vault build produces —
and drives the real `internal/scm/github.Host` against it. Against the unfixed
tree:

```
FAIL FindPR: parse gh pr list JSON: invalid character 'a' looking for beginning of value
FAIL CreatePR: URL = "automic vault: human approval required\nhttps://github.com/test/repo/pull/43" (number "43")
FAIL GetChecks poll 1: parse CI checks: invalid character 'a' looking for beginning of value
FAIL GetChecks poll 2: parse CI checks: invalid character 'a' looking for beginning of value
FAIL GetChecks poll 3: parse CI checks: invalid character 'a' looking for beginning of value
REPRODUCED: 5 gh payload(s) corrupted by a wrapper's stderr notice (issue #630)
```

and against the fixed tree the same harness prints `ok` for all five and
`NOT REPRODUCED`. That matches the reporter's three documented symptoms,
including the `CreatePR` one verbatim (the notice ends up inside `pr.URL`).

**Tests are red without the fix, green with it.** I stashed only the two source
files (`github.go`, `attachments.go`), keeping the tests:

```
--- WITHOUT FIX ---
--- FAIL: TestCreatePRKeepsURLOutOfWrapperStderrNotice
--- FAIL: TestUserAssetRepoIgnoresWrapperStderrNotice
--- FAIL: TestGetReviewCommentsIgnoresWrapperStderrNotice
--- FAIL: TestGetChecksIgnoresWrapperStderrNotice
--- FAIL: TestFindPRIgnoresWrapperStderrNotice
--- FAIL: TestGetChecksReadsCommitRollupAndRunsDespiteWrapperStderrNotice
--- WITH FIX ---
ok  github.com/kunchenguid/no-mistakes/internal/scm/github
```

**Repo gates** (Go 1.25.0, installed from `https://go.dev/dl/go1.25.0.linux-amd64.tar.gz`,
sha256 `2852af0cb20a13139b3448992e69b868e50ed0f8a1e5940ee1de9e19a123b613`):

| command | result |
| --- | --- |
| `gofmt -l .` | no output (clean) |
| `make lint` (= `go run ./cmd/genskill --check` + `go vet ./...`) | exit 0, "skills/no-mistakes/SKILL.md is up to date" |
| `go test ./internal/scm/...` | all 6 packages ok |
| `go test -race ./...` | exit 0, 52 packages ok, 0 FAIL, 4m06s wall |
| `go build -o ./bin/no-mistakes ./cmd/no-mistakes` | ok (binary removed afterwards) |
| `make e2e` | not run — see §5 |

**Both patches apply to a clean checkout of upstream `main`.** Cloned
`/work/no-mistakes` at `f4b85ae` into `/tmp/opencode/applycheck`:
`git apply --check` on the plain diff → OK; `git am --3way` on the format-patch →
OK; the resulting package test run is green and the reproduction harness reports
`NOT REPRODUCED`. So the patches, not just my working tree, are what fixes it.

**Self-review.** `go vet` clean; no `go.mod`/`go.sum`/`CHANGELOG.md`/
`.release-please-manifest.json`/generated-skill edits; working tree contains only
the four intended files. `AGENTS.md` says provider CLI traps are owned by the
comments in `internal/scm/...`, so the rationale lives in the `ghOutput` doc
comment rather than in a doc page — no user-facing behaviour changed, so no
`docs/src/content/docs/` update is warranted.

---

## 4. Sources

All retrieved **2026-10-01 (UTC)**. `api.github.com` was unusable for the whole
session (unauthenticated REST quota for this egress IP was already exhausted:
`HTTP 403`, `GET https://api.github.com/rate_limit` reported `core.remaining = 0`,
`reset = 1790833210`), so issue/PR data came from the server-rendered
`github.com` HTML payloads and code from `git clone`/`git ls-remote`.

| # | source | retrieved | used for |
| --- | --- | --- | --- |
| 1 | <https://github.com/kunchenguid/no-mistakes/issues/630> | 2026-10-01 05:10:17Z (re-fetch) | the bug: symptoms, call sites, the `ghOutput` shape, reporter `@alexlopespereira`, triage `ready-for-pr`, no linked PR |
| 2 | <https://github.com/kunchenguid/no-mistakes/issues?q=is%3Aissue+state%3Aopen+label%3Abug+sort%3Acreated-asc> | 2026-10-01 04:52:49Z, re-scraped 05:09:55Z | the 98 open `bug` issues and their creation order |
| 3 | <https://github.com/kunchenguid/no-mistakes/issues?q=is%3Apr+state%3Aopen+sort%3Acreated-asc> | 2026-10-01 | the 33 open PRs |
| 4 | <https://github.com/kunchenguid/no-mistakes> (issue pages #278…#1271, one per eligible/ineligible call) | 2026-10-01 04:53–04:55Z | labels, assignees, agent assignments, linked PRs, maintainer triage markers, claims |
| 5 | `git clone https://github.com/kunchenguid/no-mistakes.git` @ `f4b85ae` | 2026-10-01 04:52Z | `CONTRIBUTING.md`, `AGENTS.md`, `.agents/skills/testing-conventions/SKILL.md`, the code under test, all gates |
| 6 | `git ls-remote --heads https://github.com/kunchenguid/no-mistakes.git` | 2026-10-01 | branch names (no branch claims any window issue, incl. #630) |
| 7 | <https://go.dev/dl/go1.25.0.linux-amd64.tar.gz> | 2026-10-01 04:56Z | Go 1.25.0 toolchain (`go.mod` requires 1.25.0) |
| 8 | <https://github.com/kunchenguid/no-mistakes/issues/602>, `/606`, `/610`, `/620`, `/642`, `/658`, `/665`, `/667`, `/693` | 2026-10-01 04:53–04:55Z | why each other window candidate was not chosen (environment needs) |
| 9 | `https://api.github.com/rate_limit` | 2026-10-01 04:51:43Z | evidence that the REST API was unavailable (403, `remaining = 0`) |

---

## 5. Unverified / not done

- **UNVERIFIED — the "187 open issues" figure in the brief.** I measured 98 open
  issues labelled `bug` (GitHub's own `issueCount` in the search payload agrees).
  I did not independently confirm the 187 total, because that needs the REST API
  or another search page and it is only context for this pass.
- **UNVERIFIED — the claim that no *closed* PR ever attempted #630.** The issue
  page shows `linked_pull_requests: []` and `closedByPullRequestsReferences: []`,
  so no PR is linked from the issue; an unlinked, closed attempt by the reporter
  (who wrote "I have this running as a local patch … Happy to open a PR") cannot
  be ruled out from public data. I did not treat that offer as a claim: there is
  no branch, no PR, no assignee, and the reporter explicitly made it conditional
  on the maintainer asking.
- **UNVERIFIED — real-`gh` behaviour.** Every test and the reproduction drive a
  fake `gh` that mimics the documented stdout/stderr/exit-code shape. I have no
  `gh` credentials and did not run against github.com. The claim that
  `gh pr checks` reports an empty rollup on **stderr** with a non-zero exit is
  taken from the existing repo test
  `TestGetChecksIncludesFailedWorkflowRunWhenPRHasNoChecks`, which already encoded
  it, plus the reporter's note — I did not re-derive it from a live `gh`.
- **UNVERIFIED — that the patch is what the maintainer wants.** #630's newest
  triage stamp says "Preferred shape: fail fast …" for a different issue; for
  #630 it says "contract-class: restore — restore stdout-only reads for
  data-bearing `gh` invocations", which is what I implemented, but the maintainer
  has not reviewed this patch.
- **UNVERIFIED — the scope extension.** I also converted three calls the issue
  does not name (`getCommitChecks`, `getWorkflowRunChecks`,
  `GetReviewComments`) and one in `attachments.go` (`userAssetRepo`). Each is the
  same defect on a data-bearing read, each has its own regression test, and the
  maintainer's own wording ("data-bearing `gh` invocations") covers them, but that
  is my judgment call and a reviewer could reasonably ask for a narrower diff.
- **NOT DONE — `make e2e`.** Skipped deliberately: the change touches no agent
  integration, no e2e harness and no recorded fixture, and the e2e suite is behind
  the `e2e` build tag. `CONTRIBUTING.md` asks for `make e2e` "when you touch agent
  integrations, the e2e harness, or recorded fixtures" — none apply. It also needs
  real agent CLIs, which this sandbox does not have.
- **NOT DONE — the required submission path.** `CONTRIBUTING.md` requires every PR
  to be raised through `no-mistakes` itself (`no-mistakes init --fork-url …` then
  `git push no-mistakes`), and the `Require no-mistakes` workflow fails a body
  lacking the deterministic signature plus a parseable v1 pipeline attestation
  whose `head_sha` matches. That needs an authenticated agent CLI and spends
  model quota, so `/work/PR_DESCRIPTION.md` explicitly ends with a warning not to
  submit that text as-is: the `## Pipeline` section has to be written by the tool
  over the sections above it. The patch, not the prose, is the deliverable here.
- **NOT DONE — `gh` CLI.** Not installed in this sandbox; no GitHub API calls were
  made after the quota was exhausted, and no authenticated call was ever attempted.
- **ACTION REQUIRED for the submitter — placeholder commit author.** The patch was
  generated from a local commit authored as
  `no-mistakes contributor <noreply@example.invalid>`, because this work was done
  without a GitHub identity. Re-author it before pushing
  (`git commit --amend --reset-author --author "Name <email>"`); otherwise the
  published commit is attributed to a placeholder.