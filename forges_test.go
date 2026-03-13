package forges

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func assertEqual(t *testing.T, field, want, got string) {
	t.Helper()
	if want != got {
		t.Errorf("%s: want %q, got %q", field, want, got)
	}
}

func assertSliceEqual(t *testing.T, field string, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("%s: want %v, got %v", field, want, got)
		return
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("%s[%d]: want %q, got %q", field, i, want[i], got[i])
		}
	}
}

// ParseRepoURL tests

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		input               string
		domain, owner, repo string
		wantErr             bool
	}{
		{
			input:  "https://github.com/octocat/hello-world",
			domain: "github.com", owner: "octocat", repo: "hello-world",
		},
		{
			input:  "https://github.com/octocat/hello-world.git",
			domain: "github.com", owner: "octocat", repo: "hello-world",
		},
		{
			input:  "https://gitlab.com/group/project/tree/main",
			domain: "gitlab.com", owner: "group", repo: "project",
		},
		{
			input:  "github.com/user/repo",
			domain: "github.com", owner: "user", repo: "repo",
		},
		{
			input:  "git@github.com:user/repo.git",
			domain: "github.com", owner: "user", repo: "repo",
		},
		{
			input:  "git@gitlab.com:group/project.git",
			domain: "gitlab.com", owner: "group", repo: "project",
		},
		{
			input:  "https://bitbucket.org/atlassian/stash-example-plugin",
			domain: "bitbucket.org", owner: "atlassian", repo: "stash-example-plugin",
		},
		{
			input:   "",
			wantErr: true,
		},
		{
			input:   "https://github.com/just-owner",
			wantErr: true,
		},
		{
			input:   "git@github.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			domain, owner, repo, err := ParseRepoURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertEqual(t, "domain", tt.domain, domain)
			assertEqual(t, "owner", tt.owner, owner)
			assertEqual(t, "repo", tt.repo, repo)
		})
	}
}

// Client routing tests

func TestClientRouting(t *testing.T) {
	mock := &mockForge{repoService: &mockRepoService{}}
	c := NewClient(
		WithForge("github.com", mock),
		WithForge("gitlab.com", mock),
	)

	// Verify registered domains
	for _, domain := range []string{"github.com", "gitlab.com"} {
		if _, err := c.forgeFor(domain); err != nil {
			t.Errorf("expected forge for %s, got error: %v", domain, err)
		}
	}

	// Unregistered domain returns error
	_, err := c.forgeFor("example.com")
	if err == nil {
		t.Error("expected error for unregistered domain")
	}
}

func TestClientFetchRepositoryRoutes(t *testing.T) {
	mock := &mockForge{
		repoService: &mockRepoService{
			repo: &Repository{FullName: "test/repo"},
		},
	}
	c := &Client{
		forges: map[string]Forge{"example.com": mock},
		tokens: make(map[string]string),
	}

	repo, err := c.FetchRepository(context.Background(), "https://example.com/test/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.FullName != "test/repo" {
		t.Errorf("expected test/repo, got %s", repo.FullName)
	}
	ms := mock.repoService
	if ms.lastOwner != "test" || ms.lastRepo != "repo" {
		t.Errorf("expected owner=test repo=repo, got owner=%s repo=%s", ms.lastOwner, ms.lastRepo)
	}
}

func TestClientFetchTagsRoutes(t *testing.T) {
	mock := &mockForge{
		repoService: &mockRepoService{
			tags: []Tag{{Name: "v1.0.0", Commit: "abc"}},
		},
	}
	c := &Client{
		forges: map[string]Forge{"example.com": mock},
		tokens: make(map[string]string),
	}

	tags, err := c.FetchTags(context.Background(), "https://example.com/test/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}
}

// Detection tests

func TestDetectForgeTypeUsesProvidedClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-GitHub-Request-Id", "abc123")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Using a custom client should work.
	customClient := srv.Client()
	ft, err := detectFromHeaders(context.Background(), customClient, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ft != GitHub {
		t.Errorf("expected GitHub, got %s", ft)
	}
}

func TestDetectForgeTypeHeaders(t *testing.T) {
	tests := []struct {
		header string
		value  string
		want   ForgeType
	}{
		{"X-GitHub-Request-Id", "abc123", GitHub},
		{"X-Gitlab-Meta", `{"cors":"abc"}`, GitLab},
		{"X-Gitea-Version", "1.21.0", Gitea},
		{"X-Forgejo-Version", "7.0.0", Forgejo},
	}

	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(tt.header, tt.value)
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			ft, err := detectFromHeaders(context.Background(), http.DefaultClient, srv.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ft != tt.want {
				t.Errorf("want %s, got %s", tt.want, ft)
			}
		})
	}
}

func TestDetectForgeTypeGiteaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"version":"1.21.0"}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ft, err := detectFromAPI(context.Background(), http.DefaultClient, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ft != Gitea {
		t.Errorf("want Gitea, got %s", ft)
	}
}

func TestDetectForgeTypeForgejoAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"version":"7.0.0+forgejo"}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ft, err := detectFromAPI(context.Background(), http.DefaultClient, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ft != Forgejo {
		t.Errorf("want Forgejo, got %s", ft)
	}
}

func TestDetectForgeTypeGitLabAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("GET /api/v4/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"version":"16.0.0"}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ft, err := detectFromAPI(context.Background(), http.DefaultClient, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ft != GitLab {
		t.Errorf("want GitLab, got %s", ft)
	}
}

func TestDetectForgeTypeGitHubAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("GET /api/v4/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("GET /api/v3/meta", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"verifiable_password_authentication": true}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ft, err := detectFromAPI(context.Background(), http.DefaultClient, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ft != GitHub {
		t.Errorf("want GitHub, got %s", ft)
	}
}

func TestClientListRepositoriesRoutes(t *testing.T) {
	mock := &mockForge{
		repoService: &mockRepoService{
			repos: []Repository{
				{FullName: "org/repo-a"},
				{FullName: "org/repo-b"},
			},
		},
	}
	c := &Client{
		forges: map[string]Forge{"example.com": mock},
		tokens: make(map[string]string),
	}

	repos, err := c.ListRepositories(context.Background(), "example.com", "org", ListRepoOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if mock.repoService.lastOwner != "org" {
		t.Errorf("expected owner=org, got %s", mock.repoService.lastOwner)
	}
}

func TestFilterRepos(t *testing.T) {
	repos := []Repository{
		{FullName: "org/active", Archived: false, Fork: false},
		{FullName: "org/archived", Archived: true, Fork: false},
		{FullName: "org/fork", Archived: false, Fork: true},
		{FullName: "org/archived-fork", Archived: true, Fork: true},
	}

	tests := []struct {
		name string
		opts ListRepoOpts
		want []string
	}{
		{"include all", ListRepoOpts{}, []string{"org/active", "org/archived", "org/fork", "org/archived-fork"}},
		{"exclude archived", ListRepoOpts{Archived: ArchivedExclude}, []string{"org/active", "org/fork"}},
		{"only archived", ListRepoOpts{Archived: ArchivedOnly}, []string{"org/archived", "org/archived-fork"}},
		{"exclude forks", ListRepoOpts{Forks: ForkExclude}, []string{"org/active", "org/archived"}},
		{"only forks", ListRepoOpts{Forks: ForkOnly}, []string{"org/fork", "org/archived-fork"}},
		{"exclude both", ListRepoOpts{Archived: ArchivedExclude, Forks: ForkExclude}, []string{"org/active"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make([]Repository, len(repos))
			copy(input, repos)
			got := FilterRepos(input, tt.opts)
			var names []string
			for _, r := range got {
				names = append(names, r.FullName)
			}
			assertSliceEqual(t, "repos", tt.want, names)
		})
	}
}

// Mock forge for routing tests

type mockForge struct {
	repoService      *mockRepoService
	issueService     *mockIssueService
	prService        *mockPRService
	labelService     *mockLabelService
	milestoneService *mockMilestoneService
	releaseService   *mockReleaseService
	ciService        *mockCIService
	branchService    *mockBranchService
	deployKeyService *mockDeployKeyService
	secretService    *mockSecretService
	reviewService    *mockReviewService
}

func (m *mockForge) Repos() RepoService {
	return m.repoService
}

func (m *mockForge) Issues() IssueService {
	if m.issueService != nil {
		return m.issueService
	}
	return &mockIssueService{}
}

func (m *mockForge) PullRequests() PullRequestService {
	if m.prService != nil {
		return m.prService
	}
	return &mockPRService{}
}

func (m *mockForge) Labels() LabelService {
	if m.labelService != nil {
		return m.labelService
	}
	return &mockLabelService{}
}

func (m *mockForge) Milestones() MilestoneService {
	if m.milestoneService != nil {
		return m.milestoneService
	}
	return &mockMilestoneService{}
}

func (m *mockForge) Releases() ReleaseService {
	if m.releaseService != nil {
		return m.releaseService
	}
	return &mockReleaseService{}
}

func (m *mockForge) CI() CIService {
	if m.ciService != nil {
		return m.ciService
	}
	return &mockCIService{}
}

func (m *mockForge) Branches() BranchService {
	if m.branchService != nil {
		return m.branchService
	}
	return &mockBranchService{}
}

func (m *mockForge) DeployKeys() DeployKeyService {
	if m.deployKeyService != nil {
		return m.deployKeyService
	}
	return &mockDeployKeyService{}
}

func (m *mockForge) Secrets() SecretService {
	if m.secretService != nil {
		return m.secretService
	}
	return &mockSecretService{}
}

func (m *mockForge) Notifications() NotificationService {
	return &mockNotificationService{}
}

type mockNotificationService struct{}

func (m *mockNotificationService) List(_ context.Context, opts ListNotificationOpts) ([]Notification, error) {
	return nil, nil
}
func (m *mockNotificationService) MarkRead(_ context.Context, opts MarkNotificationOpts) error {
	return nil
}
func (m *mockNotificationService) Get(_ context.Context, id string) (*Notification, error) {
	return nil, nil
}

func (m *mockForge) Reviews() ReviewService {
	if m.reviewService != nil {
		return m.reviewService
	}
	return &mockReviewService{}
}

func (m *mockForge) Collaborators() CollaboratorService {
	return &mockCollaboratorService{}
}

func (m *mockForge) GetRateLimit(_ context.Context) (*RateLimit, error) {
	return nil, ErrNotSupported
}

type mockCollaboratorService struct{}

func (m *mockCollaboratorService) List(_ context.Context, _, _ string, _ ListCollaboratorOpts) ([]Collaborator, error) {
	return nil, nil
}

func (m *mockCollaboratorService) Add(_ context.Context, _, _, _ string, _ AddCollaboratorOpts) error {
	return nil
}

func (m *mockCollaboratorService) Remove(_ context.Context, _, _, _ string) error {
	return nil
}

type mockRepoService struct {
	repo      *Repository
	repos     []Repository
	tags      []Tag
	lastOwner string
	lastRepo  string
}

func (m *mockRepoService) Get(_ context.Context, owner, repo string) (*Repository, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.repo, nil
}

func (m *mockRepoService) List(_ context.Context, owner string, opts ListRepoOpts) ([]Repository, error) {
	m.lastOwner = owner
	return m.repos, nil
}

func (m *mockRepoService) Create(_ context.Context, opts CreateRepoOpts) (*Repository, error) {
	return m.repo, nil
}

func (m *mockRepoService) Edit(_ context.Context, owner, repo string, opts EditRepoOpts) (*Repository, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.repo, nil
}

func (m *mockRepoService) Delete(_ context.Context, owner, repo string) error {
	m.lastOwner = owner
	m.lastRepo = repo
	return nil
}

func (m *mockRepoService) Fork(_ context.Context, owner, repo string, opts ForkRepoOpts) (*Repository, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.repo, nil
}

func (m *mockRepoService) ListForks(_ context.Context, owner, repo string, opts ListForksOpts) ([]Repository, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.repos, nil
}

func (m *mockRepoService) ListTags(_ context.Context, owner, repo string) ([]Tag, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.tags, nil
}

func (m *mockRepoService) Search(_ context.Context, opts SearchRepoOpts) ([]Repository, error) {
	return m.repos, nil
}

type mockIssueService struct {
	issue      *Issue
	issues     []Issue
	comment    *Comment
	comments   []Comment
	lastOwner  string
	lastRepo   string
	lastNumber int
}

func (m *mockIssueService) Get(_ context.Context, owner, repo string, number int) (*Issue, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return m.issue, nil
}

func (m *mockIssueService) List(_ context.Context, owner, repo string, opts ListIssueOpts) ([]Issue, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.issues, nil
}

func (m *mockIssueService) Create(_ context.Context, owner, repo string, opts CreateIssueOpts) (*Issue, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.issue, nil
}

func (m *mockIssueService) Update(_ context.Context, owner, repo string, number int, opts UpdateIssueOpts) (*Issue, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return m.issue, nil
}

func (m *mockIssueService) Close(_ context.Context, owner, repo string, number int) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return nil
}

func (m *mockIssueService) Reopen(_ context.Context, owner, repo string, number int) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return nil
}

func (m *mockIssueService) Delete(_ context.Context, owner, repo string, number int) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return nil
}

func (m *mockIssueService) CreateComment(_ context.Context, owner, repo string, number int, body string) (*Comment, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return m.comment, nil
}

func (m *mockIssueService) ListComments(_ context.Context, owner, repo string, number int) ([]Comment, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return m.comments, nil
}

func (m *mockIssueService) ListReactions(_ context.Context, owner, repo string, number int, commentID int64) ([]Reaction, error) {
	return nil, nil
}

func (m *mockIssueService) AddReaction(_ context.Context, owner, repo string, number int, commentID int64, reaction string) (*Reaction, error) {
	return nil, nil
}

type mockPRService struct {
	pr         *PullRequest
	prs        []PullRequest
	comment    *Comment
	comments   []Comment
	diff       string
	lastOwner  string
	lastRepo   string
	lastNumber int
}

func (m *mockPRService) Get(_ context.Context, owner, repo string, number int) (*PullRequest, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return m.pr, nil
}

func (m *mockPRService) List(_ context.Context, owner, repo string, opts ListPROpts) ([]PullRequest, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.prs, nil
}

func (m *mockPRService) Create(_ context.Context, owner, repo string, opts CreatePROpts) (*PullRequest, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.pr, nil
}

func (m *mockPRService) Update(_ context.Context, owner, repo string, number int, opts UpdatePROpts) (*PullRequest, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return m.pr, nil
}

func (m *mockPRService) Close(_ context.Context, owner, repo string, number int) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return nil
}

func (m *mockPRService) Reopen(_ context.Context, owner, repo string, number int) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return nil
}

func (m *mockPRService) Merge(_ context.Context, owner, repo string, number int, opts MergePROpts) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return nil
}

func (m *mockPRService) Diff(_ context.Context, owner, repo string, number int) (string, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return m.diff, nil
}

func (m *mockPRService) CreateComment(_ context.Context, owner, repo string, number int, body string) (*Comment, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return m.comment, nil
}

func (m *mockPRService) ListComments(_ context.Context, owner, repo string, number int) ([]Comment, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return m.comments, nil
}

func (m *mockPRService) ListReactions(_ context.Context, owner, repo string, number int, commentID int64) ([]Reaction, error) {
	return nil, nil
}

func (m *mockPRService) AddReaction(_ context.Context, owner, repo string, number int, commentID int64, reaction string) (*Reaction, error) {
	return nil, nil
}

type mockLabelService struct {
	label     *Label
	labels    []Label
	lastOwner string
	lastRepo  string
	lastName  string
}

func (m *mockLabelService) List(_ context.Context, owner, repo string, opts ListLabelOpts) ([]Label, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.labels, nil
}

func (m *mockLabelService) Get(_ context.Context, owner, repo, name string) (*Label, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastName = name
	return m.label, nil
}

func (m *mockLabelService) Create(_ context.Context, owner, repo string, opts CreateLabelOpts) (*Label, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.label, nil
}

func (m *mockLabelService) Update(_ context.Context, owner, repo, name string, opts UpdateLabelOpts) (*Label, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastName = name
	return m.label, nil
}

func (m *mockLabelService) Delete(_ context.Context, owner, repo, name string) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastName = name
	return nil
}

type mockMilestoneService struct {
	milestone  *Milestone
	milestones []Milestone
	lastOwner  string
	lastRepo   string
	lastID     int
}

func (m *mockMilestoneService) List(_ context.Context, owner, repo string, opts ListMilestoneOpts) ([]Milestone, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.milestones, nil
}

func (m *mockMilestoneService) Get(_ context.Context, owner, repo string, id int) (*Milestone, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastID = id
	return m.milestone, nil
}

func (m *mockMilestoneService) Create(_ context.Context, owner, repo string, opts CreateMilestoneOpts) (*Milestone, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.milestone, nil
}

func (m *mockMilestoneService) Update(_ context.Context, owner, repo string, id int, opts UpdateMilestoneOpts) (*Milestone, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastID = id
	return m.milestone, nil
}

func (m *mockMilestoneService) Close(_ context.Context, owner, repo string, id int) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastID = id
	return nil
}

func (m *mockMilestoneService) Reopen(_ context.Context, owner, repo string, id int) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastID = id
	return nil
}

func (m *mockMilestoneService) Delete(_ context.Context, owner, repo string, id int) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastID = id
	return nil
}

type mockReleaseService struct {
	release   *Release
	releases  []Release
	asset     *ReleaseAsset
	lastOwner string
	lastRepo  string
	lastTag   string
}

func (m *mockReleaseService) List(_ context.Context, owner, repo string, opts ListReleaseOpts) ([]Release, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.releases, nil
}

func (m *mockReleaseService) Get(_ context.Context, owner, repo, tag string) (*Release, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastTag = tag
	return m.release, nil
}

func (m *mockReleaseService) GetLatest(_ context.Context, owner, repo string) (*Release, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.release, nil
}

func (m *mockReleaseService) Create(_ context.Context, owner, repo string, opts CreateReleaseOpts) (*Release, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.release, nil
}

func (m *mockReleaseService) Update(_ context.Context, owner, repo, tag string, opts UpdateReleaseOpts) (*Release, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastTag = tag
	return m.release, nil
}

func (m *mockReleaseService) Delete(_ context.Context, owner, repo, tag string) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastTag = tag
	return nil
}

func (m *mockReleaseService) UploadAsset(_ context.Context, owner, repo, tag string, _ *os.File) (*ReleaseAsset, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastTag = tag
	return m.asset, nil
}

func (m *mockReleaseService) DownloadAsset(_ context.Context, owner, repo string, _ int64) (io.ReadCloser, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return nil, nil
}

type mockCIService struct {
	run       *CIRun
	runs      []CIRun
	lastOwner string
	lastRepo  string
	lastRunID int64
}

func (m *mockCIService) ListRuns(_ context.Context, owner, repo string, opts ListCIRunOpts) ([]CIRun, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.runs, nil
}

func (m *mockCIService) GetRun(_ context.Context, owner, repo string, runID int64) (*CIRun, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastRunID = runID
	return m.run, nil
}

func (m *mockCIService) TriggerRun(_ context.Context, owner, repo string, opts TriggerCIRunOpts) error {
	m.lastOwner = owner
	m.lastRepo = repo
	return nil
}

func (m *mockCIService) CancelRun(_ context.Context, owner, repo string, runID int64) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastRunID = runID
	return nil
}

func (m *mockCIService) RetryRun(_ context.Context, owner, repo string, runID int64) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastRunID = runID
	return nil
}

func (m *mockCIService) GetJobLog(_ context.Context, owner, repo string, jobID int64) (io.ReadCloser, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return nil, nil
}

type mockBranchService struct {
	branch    *Branch
	branches  []Branch
	lastOwner string
	lastRepo  string
	lastName  string
}

func (m *mockBranchService) List(_ context.Context, owner, repo string, opts ListBranchOpts) ([]Branch, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.branches, nil
}

func (m *mockBranchService) Create(_ context.Context, owner, repo, name, from string) (*Branch, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastName = name
	return m.branch, nil
}

func (m *mockBranchService) Delete(_ context.Context, owner, repo, name string) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastName = name
	return nil
}

type mockDeployKeyService struct {
	key       *DeployKey
	keys      []DeployKey
	lastOwner string
	lastRepo  string
	lastID    int64
}

func (m *mockDeployKeyService) List(_ context.Context, owner, repo string, opts ListDeployKeyOpts) ([]DeployKey, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.keys, nil
}

func (m *mockDeployKeyService) Get(_ context.Context, owner, repo string, id int64) (*DeployKey, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastID = id
	return m.key, nil
}

func (m *mockDeployKeyService) Create(_ context.Context, owner, repo string, opts CreateDeployKeyOpts) (*DeployKey, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.key, nil
}

func (m *mockDeployKeyService) Delete(_ context.Context, owner, repo string, id int64) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastID = id
	return nil
}

type mockSecretService struct {
	secrets   []Secret
	lastOwner string
	lastRepo  string
	lastName  string
}

func (m *mockSecretService) List(_ context.Context, owner, repo string, opts ListSecretOpts) ([]Secret, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.secrets, nil
}

func (m *mockSecretService) Set(_ context.Context, owner, repo string, opts SetSecretOpts) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastName = opts.Name
	return nil
}

func (m *mockSecretService) Delete(_ context.Context, owner, repo, name string) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastName = name
	return nil
}

type mockReviewService struct {
	review     *Review
	reviews    []Review
	lastOwner  string
	lastRepo   string
	lastNumber int
}

func (m *mockReviewService) List(_ context.Context, owner, repo string, number int, opts ListReviewOpts) ([]Review, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return m.reviews, nil
}

func (m *mockReviewService) Submit(_ context.Context, owner, repo string, number int, opts SubmitReviewOpts) (*Review, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return m.review, nil
}

func (m *mockReviewService) RequestReviewers(_ context.Context, owner, repo string, number int, users []string) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return nil
}

func (m *mockReviewService) RemoveReviewers(_ context.Context, owner, repo string, number int, users []string) error {
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastNumber = number
	return nil
}
