# jitrack

Sync your git workflow with Jira tickets. Three commands automate the repetitive parts of starting, shipping, and closing a ticket — branch naming, ticket transitions, assignment, PR creation, and cross-linking between GitHub and Jira.

```sh
jitrack start KAN-123   # branch off main, assign the ticket to you, move it to In Progress
# …work, git add…
jitrack push            # commit staged changes, push, open the PR, link it on the ticket
# …PR gets reviewed and merged…
jitrack close           # move the ticket to Ready to QA, switch back to main
```

## Install

Requires Go 1.22+ and git.

```sh
make install   # builds and copies to /opt/homebrew/bin
```

## Setup

Interactive (validates credentials live):

```sh
jitrack init
```

Or non-interactive:

```sh
jitrack config set jira.url https://yourteam.atlassian.net
jitrack config set jira.email you@example.com
jitrack config set jira.token <token>     # https://id.atlassian.com/manage-profile/security/api-tokens
jitrack config set github.token <token>   # https://github.com/settings/tokens — needs Pull requests read/write and Contents read
```

Tokens can also come from the environment: `JITRACK_JIRA_TOKEN`, `JITRACK_GITHUB_TOKEN`.

## Commands

### `jitrack start TICKET-ID`

Start work on a ticket:

1. Fetches the ticket from Jira (fails fast if it doesn't exist).
2. Asks which branch prefix to use — `feature`/`fix`/`hotfix`/`chore`, preselected from the Jira issue type (`Bug` → `fix` by default, configurable via `branch_prefixes`).
3. Creates the branch off the base branch, named `<prefix>/<KEY>-<slugified-summary>`. If the branch already exists, offers to just check it out.
4. Assigns the ticket to you, moves it to *In Progress* (configurable via `transitions.start`), and comments the branch name on it. Jira hiccups here warn instead of failing — the branch is already created.

Refuses to run with uncommitted changes on tracked files.

| Option | Description |
| --- | --- |
| `--base <branch>` | Branch to base the new branch on (defaults to `base_branch` from config) |

```
$ jitrack start KAN-123
Fetching KAN-123 from Jira… ✔ [Story] Fix login redirect (To Do)
  Branch prefix for KAN-123 (Story): feature
Fetching origin… ✔
✔ created branch feature/KAN-123-fix-login-redirect from origin/main
✔ KAN-123 assigned to you
✔ KAN-123 moved to In Progress
✔ commented branch name on KAN-123

Ready to work on KAN-123 — https://yourteam.atlassian.net/browse/KAN-123
```

Branching off another branch:

```sh
jitrack start KAN-456 --base=prod   # e.g. a hotfix cut from production
```

### `jitrack push [TICKET-ID]`

Ship your work. The ticket ID is inferred from the current branch name when omitted.

1. With **staged** changes: asks for the commit type (`feat`/`fix`/`hotfix`/`chore`, preselected from the branch prefix) and the message, then commits as `<type>: KEY <message>` — e.g. `feat: KAN-123 adding new module`. The ticket key in the message enables Jira smart-commit linking.
2. With **nothing staged**: skips the commit step and just pushes your existing commits — committing by hand first is fine. Errors only if there is also nothing to push.
3. Pushes the branch to `origin` (sets upstream on first push).
4. Opens a pull request into the base branch, titled `<type>: KEY <issue summary>` — or reuses the open one, making the command safe to re-run after a failure.
5. Comments the PR link on the Jira ticket (only when a PR was actually created).

| Option | Description |
| --- | --- |
| `--base <branch>` | Branch the pull request targets (defaults to `base_branch` from config) |

```
$ git add internal/auth/session.go
$ jitrack push
  Commit type for KAN-123: feat
  Commit message (will be "feat: KAN-123 <message>"): handle expired sessions
✔ committed: 1a2b3c4 feat: KAN-123 handle expired sessions
✔ pushed feature/KAN-123-fix-login-redirect to origin
✔ pull request created: https://github.com/my-org/my-repo/pull/42
✔ commented PR link on KAN-123
```

Re-running after everything is committed (e.g. the PR step failed, or you committed manually):

```
$ jitrack push
nothing staged — pushing existing commits and ensuring PR exists
✔ pushed feature/KAN-123-fix-login-redirect to origin
✔ pull request already open: https://github.com/my-org/my-repo/pull/42
```

### `jitrack close [TICKET-ID]`

Wrap up once the PR is done. The ticket ID is inferred from the current branch name when omitted.

1. Finds the ticket's PR (by ticket key in PR branch names) and checks its state: **refuses while the PR is still open**; a PR closed without merging warns but proceeds.
2. Moves the ticket onward — default *Ready to QA*, configurable via `transitions.close`.
3. Switches your local checkout back to the base branch.

| Option | Description |
| --- | --- |
| `--base <branch>` | Branch to switch back to (defaults to `base_branch` from config) |

```
$ jitrack close
✔ PR #42 merged: https://github.com/my-org/my-repo/pull/42
✔ KAN-123 moved to Ready to QA
✔ switched to main
```

### `jitrack init`

Interactive setup wizard: prompts for the Jira URL, email, and API token, and the GitHub token — validating both credential sets with live API calls — then writes the global config file.

```
$ jitrack init
Validating Jira credentials… ✔ authenticated as Marco Avendano
Validating GitHub credentials… ✔ authenticated as MarcoAvendano
✔ config saved to ~/.config/jitrack/config.json

You're set. Try: jitrack start TICKET-123
```

### `jitrack config`

Non-interactive configuration management. Tokens are always masked in output.

| Subcommand | Description |
| --- | --- |
| `config set <key> <value>` | Set a value (unknown keys are rejected with the list of valid ones) |
| `config get <key>` | Print one effective value |
| `config list` | Print the merged config, with the source of each value (default / global / repo / env) |

| Option | Description |
| --- | --- |
| `--repo` (on `set`) | Write to the repo's `.jitrack.json` instead of the global config |

```
$ jitrack config set --repo base_branch develop
✔ base_branch set in /path/to/repo/.jitrack.json

$ jitrack config list
base_branch                    develop              (repo)
branch_prefixes.Bug            fix                  (default)
branch_prefixes.default        feature              (default)
github.api_url                 https://api.github.com (default)
github.token                   ghp_…f4Kd            (global)
jira.email                     you@example.com      (global)
jira.token                     ATAT…9fXk            (global)
jira.url                       https://yourteam.atlassian.net (global)
transitions.close              Ready to QA          (default)
transitions.start              In Progress          (default)
```

`config get` prints just the value (masked when it's a token):

```
$ jitrack config get base_branch
develop
```

## Configuration

Two JSON layers, merged (repo overrides global, env vars override both). Edit the files by hand or use `jitrack config set` — they're interchangeable.

**Global** `~/.config/jitrack/config.json` (created with mode 0600):

```json
{
  "jira":   { "url": "https://yourteam.atlassian.net", "email": "you@example.com", "token": "…" },
  "github": { "token": "…", "api_url": "https://api.github.com" }
}
```

**Per-repo** `.jitrack.json` at the repo root (optional, safe to commit — don't put tokens here). Write with `jitrack config set --repo <key> <value>`:

```json
{
  "base_branch": "main",
  "branch_prefixes": { "Bug": "bugfix", "default": "feature" },
  "transitions": { "start": "In Progress" },
  "github": { "owner": "my-org", "repo": "my-repo" }
}
```

Everything has defaults (shown above except `github.owner`/`repo`, which are auto-detected from the `origin` remote), so the tool works in any repo with zero per-repo setup.

| Key | Default | Meaning |
| --- | --- | --- |
| `base_branch` | `main` | Branch that `start` forks from and PRs target; each command's `--base` flag overrides it per run |
| `branch_prefixes.<IssueType>` | `Bug: fix`, `default: feature` | Jira issue type → branch prefix **preselected** in the `start` prompt |
| `transitions.start` | `In Progress` | Where `start` moves the ticket — matches a transition **name** ("Start work") or the **status it leads to** ("In Progress"), case-insensitive |
| `transitions.close` | `Ready to QA` | Where `close` moves the ticket once its PR is merged/closed — same name-or-status matching |
| `github.api_url` | `https://api.github.com` | Change for GitHub Enterprise |
| `github.owner` | (auto-detected from origin remote) | GitHub organization or username |
| `github.repo` | (auto-detected from origin remote) | Repository name |

## Development

```sh
make build   # go build
make test    # go test ./...
```
