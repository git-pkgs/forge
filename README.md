# forges

Go library and CLI for working with git forges. Supports GitHub, GitLab, Gitea/Forgejo, and Bitbucket Cloud through a single interface.

## CLI

```
go install github.com/git-pkgs/forge/cmd/forge@latest
```

The CLI detects which forge to use from your git remote, or you can set it with `--forge-type`.

```
forge repo view
forge issue list --state open
forge pr create --title "Fix bug" --head fix-branch
forge pr review approve 42
forge pr reviewer request 42 alice bob
forge release list
forge ci list
forge ci log 12345 --job 67890
forge branch list
forge label list
forge notification list --unread
forge notification read --id 123
forge api repos/{owner}/{repo}
```

Run `forge --help` for the full command list.

### Authentication

Store tokens with `forge auth login`:

```
forge auth login                          # interactive: asks domain + token
forge auth login --domain github.com --token ghp_abc123
forge auth login --domain gitea.example.com --token abc123 --type gitea
```

Check what's configured with `forge auth status`.

Tokens are resolved in this order: CLI flags, environment variables (`FORGE_TOKEN`, `GITHUB_TOKEN`/`GH_TOKEN`, `GITLAB_TOKEN`, `GITEA_TOKEN`, `BITBUCKET_TOKEN`), then the config file at `~/.config/forge/config`.

### Configuration

Two config files, both INI-style:

`~/.config/forge/config` stores tokens and user preferences (respects `XDG_CONFIG_HOME`):

```ini
[default]
output = json

[github.com]
token = ghp_abc123

[gitea.example.com]
type = gitea
token = abc123
```

`.forge` in the repo root is for per-project settings, committed to the repo, no tokens:

```ini
[default]
forge-type = gitlab

[gitlab.internal.dev]
type = gitlab
```

This tells forge that the project uses GitLab and that `gitlab.internal.dev` is a GitLab instance, so contributors don't each need `--forge-type` or `FORGE_HOST`.

Precedence from highest to lowest: CLI flags, environment variables, `.forge`, `~/.config/forge/config`, built-in defaults.

## Library

```go
import "github.com/git-pkgs/forge"
```

```go
client := forges.NewClient(
    forges.WithToken("github.com", os.Getenv("GITHUB_TOKEN")),
    forges.WithToken("gitlab.com", os.Getenv("GITLAB_TOKEN")),
)

repo, err := client.FetchRepository(ctx, "https://github.com/octocat/hello-world")
```

The `Forge` interface exposes services for repos, issues, pull requests, reviews, releases, CI, branches, labels, milestones, deploy keys, secrets, and notifications. Each backend implements these using its native SDK.

```go
f, _ := client.ForgeFor("github.com")
issues, _ := f.Issues().List(ctx, "octocat", "hello-world", forges.ListIssueOpts{State: "open"})
pr, _ := f.PullRequests().Get(ctx, "octocat", "hello-world", 42)
```

Self-hosted instances can be registered explicitly or detected automatically:

```go
client := forges.NewClient(
    forges.WithGitea("gitea.example.com", token),
    forges.WithGitLab("gitlab.internal.dev", token),
)

// or auto-detect
err := client.RegisterDomain(ctx, "git.example.com", token)
```

PURL support via `github.com/git-pkgs/purl`:

```go
p, _ := purl.Parse("pkg:npm/lodash?repository_url=https://github.com/lodash/lodash")
repo, err := client.FetchRepositoryFromPURL(ctx, p)
```
