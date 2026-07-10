# jitrack

Sync your git workflow with Jira tickets. Three commands automate the repetitive parts of starting, shipping, and closing a ticket:

- **`jitrack start TICKET-123`** — asks which branch prefix to use (`feature`/`fix`/`hotfix`/`chore`, preselected from the Jira issue type), creates the branch off the base branch (e.g. `feature/TICKET-123-fix-login-redirect`), assigns the ticket to you, moves it to *In Progress*, and comments the branch name on it. `--base=prod` branches off `prod` instead of the configured base.
- **`jitrack push [TICKET-123]`** — commits your **staged** changes (prompting for a message, auto-prefixed `TICKET-123:`), pushes the branch, opens a pull request into the base branch (or reuses the open one), and comments the PR link on the ticket. With nothing staged it skips the commit step and just pushes your existing commits and ensures the PR — so committing by hand first is fine. The ticket ID is inferred from the branch name when omitted. `--base=prod` targets the PR at `prod` instead of the configured base.
- **`jitrack close [TICKET-123]`** — once the ticket's PR is merged/closed (it refuses while the PR is still open), moves the ticket onward (default *Ready to QA*) and switches your local checkout back to the base branch (`--base` overrides which one). The ticket ID is inferred from the branch name when omitted.

## Install

Requires Go 1.22+ and git.

```sh
make install   # builds and copies to ~/bin (make sure it's on your PATH)
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

Inspect the effective configuration (tokens masked, source of each value shown):

```sh
jitrack config list
jitrack config get jira.url
```

## Configuration

Two JSON layers, merged (repo overrides global). Edit the files by hand or use `jitrack config set` — they're interchangeable.

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
