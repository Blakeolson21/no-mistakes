# Pass-3 selection record — kunchenguid/no-mistakes

All retrievals below happened on **2026-10-01 (UTC)** from this sandbox
(`70.120.51.9`). `api.github.com` was unusable for the whole session: the
unauthenticated REST quota for this egress IP was already exhausted
(`HTTP 403 {"message":"API rate limit exceeded for 70.120.51.9"}`,
`GET https://api.github.com/rate_limit` showed `core.remaining = 0`,
`reset = 1790833210`). Everything below therefore comes from the
**server-rendered GitHub HTML** (`github.com/...`), read out of the embedded
`react-app.embeddedData` JSON payload, plus the **git protocol**
(`git ls-remote` / `git clone`, not rate-limited). No credentials were used.

Scripts: `/work/tools/scrape_issues.py`, `/work/tools/scrape_issue_detail.py`.
Raw data: `/work/data/open-bug-issues.json`, `/work/data/open-prs.json`,
`/work/data/bug-issue-details.json`, `/work/data/eligible-order.json`.

## 1. Population

| query | GitHub-reported count | scraped |
| --- | --- | --- |
| `is:issue state:open label:bug sort:created-asc` | **98** | 98 |
| `is:pr state:open sort:created-asc` | **33** | 33 |

The task brief says "187 open issues"; the open-`bug` subset that this pass
draws from is 98 (retrieved 2026-10-01, see
`open-bug-issues.json:fetched_at`). I did **not** verify the 187 total: the
`is:issue state:open` count needs either the REST API or one more scraped
search page, and the number is only context. **UNVERIFIED: the "187 open
issues" figure in the brief.**

## 2. Eligibility filter

An issue is *ineligible* (skipped) when either holds:

1. it carries the maintainer's own `existing-pr` label, or
2. the issue page's `linkedPullRequests` contains a PR whose `state` is
   `OPEN`.

Rule 2 found exactly the same 5 issues that rule 1 found among the
`existing-pr` set; 6 further issues carry `existing-pr` with only *closed*
PRs. No open issue had an assignee and no open issue had an `agentAssignment`,
so those signals excluded nothing. Result: **87 eligible**, 11 skipped.

| skipped | created | reason |
| --- | --- | --- |
| #552 | 2026-07-23 | `existing-pr` + open linked PR #1247 |
| #924 | 2026-09-01 | `existing-pr` + open linked PR #1245 |
| #1063 | 2026-09-12 | `existing-pr` |
| #1069 | 2026-09-13 | `existing-pr` + open linked PR #1248 |
| #1139 | 2026-09-21 | `existing-pr` + open linked PR #1140 |
| #1148 | 2026-09-22 | `existing-pr` |
| #1153 | 2026-09-22 | `existing-pr` |
| #1212 | 2026-09-25 | `existing-pr` |
| #1241 | 2026-09-28 | `existing-pr` |
| #1250 | 2026-09-29 | `existing-pr` + open linked PR #1252 |
| #1251 | 2026-09-29 | `existing-pr` |

Cross-checks that found *no* additional exclusions: no remote branch name
contains any window issue number (`git ls-remote --heads origin`), and no
non-maintainer comment in the window claims the issue.

## 3. Pass-3 window — the 21st–30th oldest eligible issues

| eligible # | issue | created | reporter | subject |
| --- | --- | --- | --- | --- |
| 21 | #602 | 2026-07-28 | mrtnebrle | doctor passes without Git identity, then auto-fix commit fails |
| 22 | #606 | 2026-07-30 | mattadams-dev | PR body corruption: spliced CI log text double-encoded |
| 23 | #610 | 2026-07-30 | skillcoder | `[PRIVACY]` `NO_MISTAKES_TELEMETRY=0` ignored by managed daemon |
| 24 | #620 | 2026-08-01 | leecalcote | auto-fix commits omit Signed-off-by (DCO) |
| **25** | **#630** | **2026-08-01** | **alexlopespereira** | **gh wrappers that write to stderr corrupt FindPR, CreatePR and GetChecks** |
| 26 | #642 | 2026-08-02 | jaime-oviedo | fix-round commit amend strands the run |
| 27 | #658 | 2026-08-04 | cr101 | `agent: auto` picks unauthenticated Claude |
| 28 | #665 | 2026-08-05 | max-miller1204 | evidence SVG collapses whitespace |
| 29 | #667 | 2026-08-05 | zachlandes | PR body Intent section audience |
| 30 | #693 | 2026-08-09 | VirtualRoboticHands | custody recovery / interrupted review dead ends |

**#630 was chosen** — it is the only one in the window that reproduces on
Linux with no special hardware and no vendor credentials:

- #602 needs a real pipeline run and a git-identity-free machine (and its fix
  overlaps the open PR #1245 for #924);
- #606 needs CI log fixtures that reproduce two encoding depths, one of them
  produced by a PowerShell caret transliteration on the reporter's side;
- #610 needs an installed managed service (launchd/systemd/schtasks);
- #620 already has a PR (#621, closed) and its reporter says one is ready;
- #642 / #693 need an out-of-band history rewrite in a live gate worktree;
- #658 reproduces on Windows only, per its own report, and needs two agent
  CLIs with different auth states;
- #665 needs Chrome to see the rendering difference;
- #667 is a prompt/product-content decision, not a code defect with a failing
  test.

#630 is pure Go over a process boundary, and the repo's own
`githubTestCmdFactory` fixture already models a stderr-writing `gh` that exits
0, so the regression tests need no new machinery.

Robustness of the window: four issues carry non-`ready-for-pr` triage
outcomes (#400 `keep-open-mixed`, eligible #5; #620 `waiting-author`, #24;
#757 `help-existing-pr`, #33; #764 `needs-existing-pr`, #36). Re-running the
selection with any of those excluded keeps #630 inside positions 21–30, so the
pick does not depend on that judgment call.