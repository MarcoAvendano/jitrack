# jitrack

Sync your git workflow with Jira tickets. Three commands automate the repetitive parts of starting, shipping, and closing a ticket — branch naming, ticket transitions, assignment, PR creation, and cross-linking between your git host and Jira.

Works with **GitHub** (pull requests), **GitLab** (merge requests), and **Bitbucket** (pull requests), chosen globally at setup and overridable per-project. See [Git providers](#git-providers).

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

# GitHub (the default provider)
jitrack config set github.token <token>   # https://github.com/settings/tokens — needs Pull requests read/write and Contents read

# …or GitLab
jitrack config set provider gitlab
jitrack config set gitlab.token <token>   # https://gitlab.com/-/user_settings/personal_access_tokens — needs the 'api' scope

# …or Bitbucket (Basic auth — also needs a username)
jitrack config set provider bitbucket
jitrack config set bitbucket.username <username>   # your Bitbucket username, or Atlassian email for an API token
jitrack config set bitbucket.token <app-password>  # https://bitbucket.org/account/settings/app-passwords/ — Pull requests: Read/Write
```

Tokens can also come from the environment: `JITRACK_JIRA_TOKEN`, `JITRACK_GITHUB_TOKEN`, `JITRACK_GITLAB_TOKEN`, `JITRACK_BITBUCKET_TOKEN`.

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

Interactive setup wizard: asks which git provider you use (**GitHub**, **GitLab**, or **Bitbucket**), then prompts for the Jira URL, email, and API token, and that provider's token (plus a username for Bitbucket) — validating both credential sets with live API calls — then writes the global config file (including your `provider` choice).

```
$ jitrack init
  Git provider: gitlab
Validating Jira credentials… ✔ authenticated as Marco Avendano
Validating gitlab credentials… ✔ authenticated as marco.avendano
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
gitlab.api_url                 https://gitlab.com/api/v4 (default)
jira.email                     you@example.com      (global)
jira.token                     ATAT…9fXk            (global)
jira.url                       https://yourteam.atlassian.net (global)
provider                       github               (default)
transitions.close              Ready to QA          (default)
transitions.start              In Progress          (default)
```

`config get` prints just the value (masked when it's a token):

```
$ jitrack config get base_branch
develop
```

## Git providers

jitrack talks to a git host through a small provider abstraction, so `start`, `push`, and `close` behave identically no matter where your code lives. Three providers ship today:

| Provider | `provider` value | What a "PR" is | Auth |
| --- | --- | --- | --- |
| GitHub *(default)* | `github` | Pull request | Fine-grained PAT: Pull requests read/write **and** Contents read |
| GitLab | `gitlab` | Merge request | PAT with the `api` scope |
| Bitbucket | `bitbucket` | Pull request | Username + app password (Pull requests: Read/Write) via HTTP Basic auth |

The active provider is the `provider` config key (defaults to `github`). Because it's an ordinary config key, it follows the normal layering — **set it globally at `init`, override it per-project** in the repo's `.jitrack.json`:

```sh
# global default for all repos
jitrack config set provider github

# this repo lives on GitLab — override just here
jitrack config set provider gitlab --repo
jitrack config set gitlab.token <token> --repo   # or keep the token global / in the env
```

`config list` shows which layer won:

```
$ jitrack config list | grep provider
provider                       gitlab               (repo)
```

Each provider reads its own namespaced keys — `github.token`/`github.api_url`/`github.owner`/`github.repo` and the matching `gitlab.*` / `bitbucket.*`. Owner/repo (for GitLab, the full `group/subgroup/project` path; for Bitbucket, the `workspace`/`repo-slug`) are auto-detected from the `origin` remote when not set. Point `*.api_url` at your own host for GitHub Enterprise or a self-managed instance. Bitbucket additionally needs `bitbucket.username` because it authenticates with HTTP Basic auth rather than a single bearer token.

**Adding another provider** (Gitea, Azure DevOps, …) is a contained change: add a transport client under `internal/<name>/`, an adapter in `internal/forge/`, one `switch` case in `forge.New`, and the config keys — no changes to the commands.

## Configuration

Two JSON layers, merged (repo overrides global, env vars override both). Edit the files by hand or use `jitrack config set` — they're interchangeable.

**Global** `~/.config/jitrack/config.json` (created with mode 0600):

```json
{
  "provider": "github",
  "jira":   { "url": "https://yourteam.atlassian.net", "email": "you@example.com", "token": "…" },
  "github": { "token": "…", "api_url": "https://api.github.com" }
}
```

For GitLab, set `"provider": "gitlab"` and a `"gitlab": { "token": "…" }` block instead.

**Per-repo** `.jitrack.json` at the repo root (optional, safe to commit — don't put tokens here). Write with `jitrack config set --repo <key> <value>`:

```json
{
  "base_branch": "main",
  "branch_prefixes": { "Bug": "bugfix", "default": "feature" },
  "transitions": { "start": "In Progress" },
  "provider": "gitlab",
  "gitlab": { "owner": "my-group", "repo": "my-repo" }
}
```

Everything has defaults (shown above except `*.owner`/`*.repo`, which are auto-detected from the `origin` remote), so the tool works in any repo with zero per-repo setup.

| Key | Default | Meaning |
| --- | --- | --- |
| `provider` | `github` | Git host: `github`, `gitlab`, or `bitbucket`. Override per-repo with `config set provider … --repo` — see [Git providers](#git-providers) |
| `base_branch` | `main` | Branch that `start` forks from and PRs target; each command's `--base` flag overrides it per run |
| `branch_prefixes.<IssueType>` | `Bug: fix`, `default: feature` | Jira issue type → branch prefix **preselected** in the `start` prompt |
| `transitions.start` | `In Progress` | Where `start` moves the ticket — matches a transition **name** ("Start work") or the **status it leads to** ("In Progress"), case-insensitive |
| `transitions.close` | `Ready to QA` | Where `close` moves the ticket once its PR is merged/closed — same name-or-status matching |
| `github.api_url` | `https://api.github.com` | Change for GitHub Enterprise |
| `github.owner` / `github.repo` | (auto-detected from origin remote) | GitHub organization/username and repository |
| `gitlab.api_url` | `https://gitlab.com/api/v4` | Change for a self-managed GitLab instance |
| `gitlab.owner` / `gitlab.repo` | (auto-detected from origin remote) | GitLab namespace (group/subgroup) and project |
| `bitbucket.username` | (none) | Bitbucket username or Atlassian email — required for Basic auth |
| `bitbucket.api_url` | `https://api.bitbucket.org/2.0` | Change for a self-managed Bitbucket instance |
| `bitbucket.owner` / `bitbucket.repo` | (auto-detected from origin remote) | Bitbucket workspace and repository slug |

## Known issues

Most surprises come from configuration or credentials rather than bugs. The common ones:

### `start` fails with `origin/main is not a commit`

```
Error: git checkout -b feature/… origin/main --no-track: fatal: 'origin/main' is not a commit …
```

Your repo's default branch isn't `main` (GitLab and older repos often use `master`). `base_branch` defaults to `main`. Point it at the right branch — per-repo:

```sh
jitrack config set base_branch master --repo   # or --base master for a one-off, or drop --repo to set it globally
```

`base_branch` is what `start` branches from, what `push` targets, and what `close` returns to — so setting it once fixes all three.

### `unknown config key "provider"` (or `gitlab.*`, `bitbucket.*`)

```
Error: unknown config key "provider" — valid keys: jira.url, jira.email, …
```

The `jitrack` on your `PATH` is an **older build** that predates multi-provider support. `make build` only updates the local `./jitrack`; reinstall the version on your `PATH`:

```sh
make install   # rebuilds and copies to /opt/homebrew/bin
```

Re-run `make install` after every update, not just `make build`.

### GitHub: PR creation fails with HTTP 422 "not all refs are readable"

A fine-grained PAT needs **Contents: Read** in addition to **Pull requests: Read/Write**. Listing PRs works without it, so `push` gets all the way to the create step before failing. Add the Contents: Read permission to the token.

### GitLab: merge-request creation returns 403 / "insufficient scope"

The token needs the **`api`** scope (read *and* write). A read-only `read_api` token can list MRs but not create them. Regenerate at `https://gitlab.com/-/user_settings/personal_access_tokens` with `api` checked.

For a **self-managed** instance, `gitlab.api_url` must include the API path — e.g. `https://gitlab.example.com/api/v4`, not just the web URL.

### Bitbucket: `authentication failed — check bitbucket.username and bitbucket.token`

Bitbucket Cloud uses **HTTP Basic auth**, so it needs both `bitbucket.username` *and* `bitbucket.token` — a token alone won't authenticate. Use your Bitbucket username (or Atlassian email for an API token) plus an **app password** with Pull requests: Read/Write. Workspace/repo *access tokens* (bearer) are not supported because they can't validate via `/user`.

### Jira says the ticket doesn't exist, but it does

Jira returns **404 (not 401)** for a bad token on issue fetch, so "issue not found" during `start`/`push` often means the API token is wrong or expired, not that the key is invalid. Re-check `jira.token` / `jira.email` (or run `jitrack init`).

### Ticket doesn't move: `could not move ticket to "…"`

Jira only offers transitions valid from the ticket's **current** status. If `transitions.start` (default `In Progress`) or `transitions.close` (default `Ready to QA`) doesn't match an available transition, jitrack **warns and continues** — the git work still succeeds. Fixes:

- Make the target match a transition **name** or the **status it leads to** (matching is case-insensitive): `jitrack config set transitions.close "In Review" --repo`.
- Make sure the ticket is in a status from which that transition is allowed by your board's workflow.

### Don't put tokens in `.jitrack.json`

`config set … --repo` writes to `.jitrack.json`, which is meant to be committed. Never set `*.token` with `--repo` — keep tokens in the global config (`~/.config/jitrack/config.json`, mode 0600) or the environment (`JITRACK_*_TOKEN`).

### `push`: "nothing staged and no commits to push"

`push` commits only **staged** changes. With nothing staged and no commits ahead of the base branch, there's nothing to do — `git add` your changes first (or commit by hand, then `push` will just push and open the PR).

## Development

```sh
make build   # go build
make test    # go test ./...
```

## License

Released under the [MIT License](LICENSE) — free to use, modify, and distribute.
