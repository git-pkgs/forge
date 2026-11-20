# forge

Go library and CLI for working with git forges. Supports GitHub, GitLab, Gitea/Forgejo, Bitbucket Cloud, Gerrit, Tangled and Radicle (read-only) through a single interface.

## CLI

```
brew install git-pkgs/git-pkgs/forge
```

Or with Go:

```
go install github.com/git-pkgs/forge/cmd/forge@latest
```

The CLI detects which forge to use from your git remote, or you can set it with `--forge-type`. Upstream is built in: remotes on `git.coline.app` and `ssh.git.coline.app`, where Upstream serves git, resolve to its API on `upstream.codes`.

```
forge repo view
forge issue list --state open
forge pr create --title "Fix bug" --head fix-branch
forge pr create --fill
forge pr review approve 42
forge pr reviewer request 42 alice bob
forge release list
forge ci list
forge ci log 67890
forge branch list
forge label list
forge webhook add --url https://example.com/hook -e push -e pull_request
forge notification list --unread
forge notification read --id 123
forge attestation download ./forge_linux_amd64.tar.gz
forge api repos/{owner}/{repo}
```

Run `forge --help` for the full command list.

### Authentication

Store tokens with `forge auth login`:

```
forge auth login                          # interactive: asks domain + token
forge auth login --domain github.com --token ghp_abc123
forge auth login --domain gitea.example.com --token abc123 --type gitea
forge auth login --domain gerrit.example.com --token user:http_password --type gerrit
forge auth login --domain github.com --token-cmd 'rbw get github-token'
```

`--token-cmd` stores a shell command instead of a literal token; the command
is run each time the token is needed (see [token commands](#token-commands) below).

When prompted for a token interactively, press **Ctrl+E** as the first key
to enter a command instead:

```
Token for github.com (Ctrl+E first for command):
Command for token (e.g. rbw get github.com): rbw get github-token
```

Check what's configured with `forge auth status`.

Tokens are resolved in this order: CLI flags, environment variables (`FORGE_TOKEN`, `GITHUB_TOKEN`/`GH_TOKEN`, `GITLAB_TOKEN`, `FORGEJO_TOKEN`/`GITEA_TOKEN`, `BITBUCKET_TOKEN`, `TANGLED_TOKEN`, `UPSTREAM_TOKEN`), then the config file at `~/.config/forge/config`. The target host is inferred from the current directory's git remote; use `--host` to override it (for example `forge --host gitea.com repo list someone`). `FORGE_HOST` sets the host for `-R OWNER/REPO` and for commands that don't act on the current repo, but doesn't override the current repo's remote.

Radicle repositories have no owner, so they're selected by repository ID: `-R rad:<RID>` or `-R HOST/rad:<RID>`. Inside a clone made with `rad clone` or `rad init`, the ID is read from its `rad://` remote, so `-R` isn't needed. That remote is usually called `rad`, and it's used automatically when there's no `origin`. If the clone also has an `origin`, pick the Radicle remote with `--remote rad` (for example `forge --remote rad --host seed.radicle.dev repo view`). The node to query comes from the host in `-R`, `--host`, `FORGE_HOST` or a section with `type = radicle` in your user config (a project `.forge` file can't pick the node). There's no default node, so a repository ID is never sent to a seed you didn't choose. For example `forge --host seed.radicle.dev -R rad:z3gqcJUoA1n9HaHKufZs5FCSGazv5 file ls`. The Radicle backend is read-only over `radicle-httpd` and currently covers repositories, branches, tags and files.

### Configuration

Two config files, both INI-style:

`~/.config/forge/config` stores tokens and user preferences (respects `XDG_CONFIG_HOME`):

```ini
[default]
output = json
remote = origin

[github.com]
token = ghp_abc123

[gitea.example.com]
type = gitea
token = abc123

[gerrit.example.com]
type = gerrit
token = user:http_password
```

### Token commands

Instead of a literal `token`, use `token-cmd` to specify a shell command.
The command is executed via `sh -c` (Unix) or `%COMSPEC% /c` (Windows) each
time forge needs the token and its stdout is used as the value. This lets you
fetch secrets from a password manager instead of storing them in plain text:

```ini
[github.com]
token-cmd = rbw get github-token

[gitlab.com]
token-cmd = pass show forge/gitlab

[myhostedgitlab.example.com]
token-cmd = rbw get --raw myhostedgitlab | jq -r '.fields | map(select(.name == "token"))[0].value'
```

The variable `FORGE_DOMAIN` is set to the domain name when the command runs
(reference it as `$FORGE_DOMAIN` on Unix, `%FORGE_DOMAIN%` on Windows), so a
single command can serve multiple domains:

```ini
[github.com]
token-cmd = pass show forge/$FORGE_DOMAIN

[myhostedgitlab.example.com]
token-cmd = pass show forge/$FORGE_DOMAIN
```

`forge auth login` sets this up interactively (Ctrl+E at the token prompt).
`forge auth status` shows the command source instead of the resolved value.

`.forge` in the repo root is for per-project settings, committed to the repo, no tokens:

```ini
[default]
forge-type = gitlab
remote = devrepo

[gitlab.internal.dev]
type = gitlab
```

This tells forge to use the `devrepo` git remote, that the project uses GitLab, and that `gitlab.internal.dev` is a GitLab instance, so contributors don't each need `--remote`, `--forge-type`, or `FORGE_HOST`.

The `[default]` `remote` setting selects the git remote used when `--remote` is omitted. If neither is set, forge uses `origin`.

For a self-hosted instance served over plain HTTP (a local Forgejo in Docker, say), add `scheme = http` to its section in `~/.config/forge/config`, use `forge auth login --scheme http`, or pass a full URL to `--host`/`FORGE_HOST`:

```ini
[172.30.0.10:3000]
type = forgejo
scheme = http
```

Committed `.forge` files cannot set the API scheme.

Precedence from highest to lowest: CLI flags, environment variables, `.forge`, `~/.config/forge/config`, built-in defaults.

### Read-only mode

`--read-only`, or `FORGE_READ_ONLY=true` in the environment, stops forge from changing anything on the forge. Only GET, HEAD and OPTIONS requests are sent, so listing and viewing work as usual, and any command that would create, edit or delete something fails before its request leaves the machine. This covers `forge api` too, and `pr create --push` is refused before anything is pushed. It's meant for CI jobs, monitoring and audits that need a token but should never write with it.

```
FORGE_READ_ONLY=true forge issue list
forge --read-only label create bug   # Error: create label: Post "...": write refused in read-only mode
```

`--read-only=false` overrides `FORGE_READ_ONLY` for a single command.

### Attestations

`forge attestation` finds and downloads the artifact attestations, such as SLSA provenance and SBOMs signed as Sigstore bundles, that a repository holds for an artifact on GitHub or GitLab. Pass the artifact's path, which is hashed with SHA-256, or its digest as `sha256:<hex>`:

```
forge attestation list ./forge_linux_amd64.tar.gz
forge attestation list sha256:5db1fee4b5703808c48078a76768b155b421b210c0761cd6a5d223f4d99f1eaa --predicate-type provenance
forge attestation download ./forge_linux_amd64.tar.gz
forge attestation download ./forge_linux_amd64.tar.gz -O bundles.jsonl
```

`download` writes every bundle as JSON Lines, one per line, to `sha256:<hex>.jsonl` (`sha256-<hex>.jsonl` on Windows) unless `-O` names another file, the same layout `gh attestation download` uses, so the file can be checked with `gh attestation verify --bundle`. Nothing is written if any bundle fails to download. forge doesn't verify bundles itself. `--predicate-type` takes a predicate type URI or `provenance` or `sbom`, and `--limit` caps how many attestations are fetched. GitHub doesn't report predicate types when listing, so on GitHub `list` reads each one from the attestation's bundle.

## Library

```sh
go get github.com/git-pkgs/forge
```

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

The `Forge` interface exposes services for repos, issues, pull requests, reviews, releases, CI, branches, labels, milestones, deploy keys, webhooks, secrets, notifications, files, collaborators, commit statuses, commits and artifact attestations. Each backend implements these using its native SDK.

On Bitbucket Cloud, secrets are the repository's Bitbucket Pipelines variables. Listing returns every variable, secured or not, and setting one stores a secured variable, updating and securing any existing variable with that name. Bitbucket deploy keys can only read, so creating one with `ReadOnly` unset (`forge deploy-key add --read-only=false`) returns `forges.ErrNotSupported`.

Under TinyGo, `gitlab.New` retains its signature and returns a backend whose API operations report `forges.ErrNotSupported`. GitLab URL parsing, web link builders and webhook parsing remain available. Native Go retains the full GitLab implementation.

```go
f, _ := client.ForgeFor("github.com")
issues, _ := f.Issues().List(ctx, "octocat", "hello-world", forges.ListIssueOpts{State: "open"})
pr, _ := f.PullRequests().Get(ctx, "octocat", "hello-world", 42)
```

Failures map to common errors on every forge, so callers can check the kind of failure without knowing which forge they're using. Apart from `ErrNotFound`, which is returned on its own, the original error stays in the chain for callers that need forge-specific details:

```go
_, err := f.Repos().Get(ctx, "octocat", "hello-world")
switch {
case errors.Is(err, forges.ErrNotFound):      // 404
case errors.Is(err, forges.ErrUnauthorized):  // 401 or 403
case errors.Is(err, forges.ErrRateLimited):   // 429, or GitHub's 403 rate limit responses
case errors.Is(err, forges.ErrServerError):   // 5xx
}
```

The `factory` package imports every provider and returns a `Client` that already knows github.com, gitlab.com, codeberg.org, bitbucket.org, tangled.org and upstream.codes, and sends repository URLs on Upstream's git hosts, `git.coline.app` and `ssh.git.coline.app`, to upstream.codes. `forges.WithHostAlias` adds such a mapping for another forge. It detects and registers any other domain over https the first time a repository URL on it is used, which sends requests to that domain:

```go
import "github.com/git-pkgs/forge/factory"

client := factory.NewClient(forges.WithToken("git.example.com", token))

f, owner, repo, err := client.ForgeForURL(ctx, "https://git.example.com/octocat/hello-world")
if err != nil {
    return err
}
issues, err := f.Issues().List(ctx, owner, repo, forges.ListIssueOpts{State: "open"})
```

Detection never uses plain HTTP, so an http-only instance has to be registered with `RegisterDomain` before its URLs are used:

```go
if err := client.RegisterDomain(ctx, "http://git.internal.example", token, factory.Builders()); err != nil {
    return err
}
```

Incoming webhook deliveries from GitHub, GitLab and Gitea/Forgejo can be verified and parsed into a normalized `WebhookEvent`. Event types use the same names as `Webhook.Events`. Push, pull request, issue, comment and release deliveries carry typed fields. The raw JSON is always available in `Payload`.

An empty secret skips verification, so read the secret once at startup and refuse to run without it:

```go
secret := os.Getenv("WEBHOOK_SECRET")
if secret == "" {
    log.Fatal("WEBHOOK_SECRET must be set")
}

http.HandleFunc("/hook", func(w http.ResponseWriter, r *http.Request) {
    event, err := f.Webhooks().ParseEvent(r, secret)
    if errors.Is(err, forges.ErrInvalidWebhookSignature) {
        http.Error(w, "bad signature", http.StatusUnauthorized)
        return
    }
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if event.Type == "pull_request" && event.Action == "opened" {
        log.Printf("new PR #%d in %s", event.PullRequest.Number, event.Repository.FullName)
    }
})
```

Each backend also exports `ParseWebhook(r, secret)` (for example `github.ParseWebhook`), which needs no API client.

Artifact attestations, such as SLSA provenance and SBOMs signed as Sigstore bundles, can be looked up by repository and artifact digest on GitHub and GitLab. `List` follows pagination and returns metadata for every attestation; `Download` then fetches each original bundle without verifying it. An empty result means the lookup succeeded and found nothing. Request failures are returned as errors, and other backends return `forges.ErrNotSupported`:

```go
digest := "sha256:5db1fee4b5703808c48078a76768b155b421b210c0761cd6a5d223f4d99f1eaa"
atts, err := client.ListAttestations(ctx, "https://github.com/cli/cli", digest, forges.ListAttestationOpts{})
if err != nil {
    log.Fatal(err)
}
for _, att := range atts {
    bundle, err := client.DownloadAttestation(ctx, "https://github.com/cli/cli", att)
    if err != nil {
        log.Fatal(err)
    }
    // verify bundle with a Sigstore client
}
```

`ListAttestationOpts.PredicateType` filters by predicate type URI or by the short names `provenance` and `sbom`. GitLab's attestation API is an Ultimate-tier experiment behind the `slsa_provenance_statement` feature flag, and GitLab answers 404 when it is off, so `List` reports `forges.ErrNotFound` for a project without it.

Self-hosted instances can be registered explicitly or detected automatically:

```go
import (
    "github.com/git-pkgs/forge/gitea"
    "github.com/git-pkgs/forge/gitlab"
    "github.com/git-pkgs/forge/radicle"
    "github.com/git-pkgs/forge/tangled"
)

client := forges.NewClient(
    forges.WithForge("gitea.example.com", gitea.New("https://gitea.example.com", token, nil)),
    forges.WithForge("gitlab.internal.dev", gitlab.New("https://gitlab.internal.dev", token, nil)),
)

// or auto-detect the forge type
err := client.RegisterDomain(ctx, "git.example.com", token, forges.ForgeBuilders{
    GitHub: github.NewWithBase,
    GitLab: gitlab.New,
    Gitea:  gitea.New,
    Tangled: tangled.New,
    Radicle: radicle.New,
})
```

For read-only access, wrap the HTTP client with `forges.ReadOnlyHTTPClient` and give it to every backend you construct and to `WithHTTPClient`. Requests other than GET, HEAD and OPTIONS then fail with `forges.ErrReadOnly` without being sent, whichever backend or operation made them:

```go
hc := forges.ReadOnlyHTTPClient(nil)
client := forges.NewClient(
    forges.WithHTTPClient(hc),
    forges.WithForge("github.com", github.New(token, hc)),
)

f, _ := client.ForgeFor("github.com")
_, err := f.Issues().Create(ctx, "octocat", "hello-world", forges.CreateIssueOpts{Title: "x"})
errors.Is(err, forges.ErrReadOnly) // true
```

Credentials that a fixed token can't express, such as a GitHub App installation token that expires after an hour or a token that comes from a secrets manager, go in the HTTP client instead. `forges.AuthHTTPClient` returns a copy of a client that passes every request to a `forges.Credentials` before sending it. Give it to each backend with an empty token, so the backend adds no credentials of its own, and to `WithHTTPClient`. `forges.BearerToken` covers GitHub, GitLab, Gitea/Forgejo, Bitbucket Cloud and Tangled. It asks its `TokenSource` for a token on every request and only sends it to the named host and its subdomains, so `github.com` covers `api.github.com` but detection probes and attestation bundle downloads on other hosts get no token:

```go
ts := forges.TokenSourceFunc(func(ctx context.Context) (string, error) {
    return installationTokens.Get(ctx) // cached, refreshed before it expires
})
hc := forges.AuthHTTPClient(nil, forges.BearerToken("github.com", ts))

client := factory.NewClient(forges.WithHTTPClient(hc))
```

For anything else, such as a different token per organization or signing each request, implement `Credentials` or use `forges.CredentialsFunc`. It gets a copy of every request the client sends, to any host, so check `req.URL` before adding anything. An error stops the request from being sent. Gerrit only uses its authenticated endpoints when it has a token, so it needs `gerrit.New` with a token instead.

```go
hc := forges.AuthHTTPClient(nil, forges.CredentialsFunc(func(req *http.Request) error {
    if req.URL.Host != "git.example.com" {
        return nil
    }
    owner, _, _ := strings.Cut(strings.TrimPrefix(req.URL.Path, "/api/v1/repos/"), "/")
    if token, ok := orgTokens[owner]; ok {
        req.Header.Set("Authorization", "Bearer "+token)
    }
    return nil
}))
f := gitea.New("https://git.example.com", "", hc)
```

`AuthHTTPClient` and `ReadOnlyHTTPClient` can be combined: `forges.ReadOnlyHTTPClient(forges.AuthHTTPClient(nil, creds))`.

PURL support via `github.com/git-pkgs/purl`:

```go
p, _ := purl.Parse("pkg:npm/lodash?repository_url=https://github.com/lodash/lodash")
repo, err := client.FetchRepositoryFromPURL(ctx, p)
```

Branches, tags and abbreviated commit refs can be resolved to full commit SHAs without listing every tag. GitHub, GitLab and Gitea/Forgejo implement this; the remaining backends return `forges.ErrNotSupported`.

```go
f, _ := client.ForgeFor("gitlab.com")
sha, err := f.Commits().ResolveCommit(ctx, "gitlab-org", "gitlab", "v17.0.0")

// or route by repository URL
sha, err = client.ResolveCommit(ctx, "https://github.com/actions/checkout", "v4.2.1")
```

A ref that does not exist returns `forges.ErrNotFound`. A ref that is already a full 40-character SHA is normalized and returned without a network request.

Funding links live in a `FUNDING.yml` file, and the `funding` package looks for it where GitHub does: `.github/`, the repository root, `docs/`, then the owner's `.github` repository.

```go
import "github.com/git-pkgs/forge/funding"

f, _ := client.ForgeFor("github.com")
links, err := funding.Discover(ctx, f.Files(), "octocat", "hello-world", "")
// links.GitHub holds ["sponsors-handle"], links.Custom the URLs the file lists.

// A caller that already has the file parses it without a forge:
links, err = funding.Parse(body)
```

The GitHub backend also exposes a standalone resolver for callers that do not need the full `Forge` interface:

```go
import githubforge "github.com/git-pkgs/forge/github"

resolver := githubforge.NewCommitResolver(os.Getenv("GITHUB_TOKEN"), nil)
sha, err := resolver.ResolveCommit(ctx, "actions", "checkout", "v4.2.1")
```

## Testing

```sh
go test ./...
```

Most backend tests run against `httptest` servers with hand-written responses. Tests named `TestReplay*` instead replay responses recorded from the real APIs (api.github.com, gitlab.com, codeberg.org and api.bitbucket.org), so they check the backends against what the forges actually send without needing network access. The recordings live in `testdata/replay/<test name>.json` next to each test.

Set `FORGE_RECORD=1` to send the requests to the real API and rewrite the recordings, for example after adding a test or when an API changes:

```sh
FORGE_RECORD=1 go test ./github -run TestReplayGetRepo
```

A recording is only written when the test passes. Only the method, URL and body of each request are stored, so a token sent in a request header never ends up in a fixture, and only the response headers the SDKs read (`Content-Type`, `Link`, pagination and `X-Gitlab-*` file headers) are kept. Replay fails the test if the code makes a request that is not in the recording, or if a recorded request is never made.

New replay tests get their client from `internal/httpreplay`:

```go
f := github.New("", httpreplay.Client(t))
repo, err := f.Repos().Get(ctx, "octocat", "Hello-World")
```

## License

[MIT](LICENSE).
