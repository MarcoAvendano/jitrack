# sr-cli

Sync your git workflow with Jira tickets. Two commands cover the repetitive parts of starting and shipping a ticket:

- **`sr-cli start SR-123`** — asks which branch prefix to use (`feature`/`fix`/`hotfix`/`chore`, preselected from the Jira issue type), creates the branch off the base branch (e.g. `feature/SR-123-fix-login-redirect`), moves the ticket to *In Progress*, and comments the branch name on it.
- **`sr-cli push [SR-123]`** — commits your **staged** changes (prompting for a message, auto-prefixed `SR-123:`), pushes the branch, opens a pull request into the base branch (or reuses the open one), and comments the PR link on the ticket. The ticket ID is inferred from the branch name when omitted.

## Install

Requires Go 1.22+ and git.

```sh
make install   # builds and copies to ~/bin (make sure it's on your PATH)
```

## Setup

Interactive (validates credentials live):

```sh
sr-cli init
```

Or non-interactive:

```sh
sr-cli config set jira.url https://yourteam.atlassian.net
sr-cli config set jira.email you@example.com
sr-cli config set jira.token <token>     # https://id.atlassian.com/manage-profile/security/api-tokens
sr-cli config set github.token <token>   # https://github.com/settings/tokens — needs Pull requests read/write and Contents read
```

Tokens can also come from the environment: `SR_JIRA_TOKEN`, `SR_GITHUB_TOKEN`.

Inspect the effective configuration (tokens masked, source of each value shown):

```sh
sr-cli config list
sr-cli config get jira.url
```

## Configuration

Two JSON layers, merged (repo overrides global). Edit the files by hand or use `sr-cli config set` — they're interchangeable.

**Global** `~/.config/sr-cli/config.json` (created with mode 0600):

```json
{
  "jira":   { "url": "https://yourteam.atlassian.net", "email": "you@example.com", "token": "…" },
  "github": { "token": "…", "api_url": "https://api.github.com" }
}
```

**Per-repo** `.sr-cli.json` at the repo root (optional, safe to commit — don't put tokens here). Write with `sr-cli config set --repo <key> <value>`:

```json
{
  "base_branch": "main",
  "branch_prefixes": { "Bug": "bugfix", "default": "feature" },
  "transitions": { "start": "In Progress" },
  "github": { "owner": "Team-Storyrocket", "repo": "storyrocket-react" }
}
```

Everything has defaults (shown above except `github.owner`/`repo`, which are auto-detected from the `origin` remote), so the tool works in any repo with zero per-repo setup.

| Key | Default | Meaning |
| --- | --- | --- |
| `base_branch` | `main` | Branch that `start` forks from and PRs target |
| `branch_prefixes.<IssueType>` | `Bug: fix`, `default: feature` | Jira issue type → branch prefix **preselected** in the `start` prompt |
| `transitions.start` | `In Progress` | Where `start` moves the ticket — matches a transition **name** ("Start work") or the **status it leads to** ("In Progress"), case-insensitive |
| `github.api_url` | `https://api.github.com` | Change for GitHub Enterprise |

## Development

```sh
make build   # go build
make test    # go test ./...
```
