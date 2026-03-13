package forges

import "time"

// ForgeType identifies which forge software a domain runs.
type ForgeType string

const (
	GitHub    ForgeType = "github"
	GitLab    ForgeType = "gitlab"
	Gitea     ForgeType = "gitea"
	Forgejo   ForgeType = "forgejo"
	Bitbucket ForgeType = "bitbucket"
	Unknown   ForgeType = "unknown"
)

// User holds normalized user/org metadata.
type User struct {
	Login     string `json:"login"`
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	HTMLURL   string `json:"html_url,omitempty"`
	IsOrg     bool   `json:"is_org,omitempty"`
}

// Repository holds normalized metadata about a source code repository,
// independent of which forge hosts it.
type Repository struct {
	FullName            string    `json:"full_name"`
	Owner               string    `json:"owner"`
	Name                string    `json:"name"`
	Description         string    `json:"description,omitempty"`
	Homepage            string    `json:"homepage,omitempty"`
	HTMLURL             string    `json:"html_url"`
	CloneURL            string    `json:"clone_url,omitempty"`
	SSHURL              string    `json:"ssh_url,omitempty"`
	Language            string    `json:"language,omitempty"`
	License             string    `json:"license,omitempty"` // SPDX identifier
	DefaultBranch       string    `json:"default_branch,omitempty"`
	Fork                bool      `json:"fork"`
	Archived            bool      `json:"archived"`
	Private             bool      `json:"private"`
	MirrorURL           string    `json:"mirror_url,omitempty"`
	SourceName          string    `json:"source_name,omitempty"` // fork parent full name
	Size                int       `json:"size"`
	StargazersCount     int       `json:"stargazers_count"`
	ForksCount          int       `json:"forks_count"`
	OpenIssuesCount     int       `json:"open_issues_count"`
	SubscribersCount    int       `json:"subscribers_count"`
	HasIssues           bool      `json:"has_issues"`
	PullRequestsEnabled bool      `json:"pull_requests_enabled"`
	Topics              []string  `json:"topics,omitempty"`
	LogoURL             string    `json:"logo_url,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	PushedAt            time.Time `json:"pushed_at,omitzero"`
}

// Tag represents a git tag.
type Tag struct {
	Name   string `json:"name"`
	Commit string `json:"commit"` // SHA
}

// ArchivedFilter controls how archived repositories are handled in list operations.
type ArchivedFilter int

const (
	ArchivedInclude ArchivedFilter = iota
	ArchivedExclude
	ArchivedOnly
)

// ForkFilter controls how forked repositories are handled in list operations.
type ForkFilter int

const (
	ForkInclude ForkFilter = iota
	ForkExclude
	ForkOnly
)

// Visibility selects the visibility level for a new or edited repository.
type Visibility int

const (
	VisibilityDefault Visibility = iota
	VisibilityPublic
	VisibilityPrivate
	VisibilityInternal
)

// ListRepoOpts configures a repo list call.
//
// Pagination: Page and PerPage control the API page size and starting page.
// Limit caps the total number of results returned across all pages. When
// Limit is 0 all results are returned. PerPage defaults to a backend-specific
// value (typically 30-50) when 0.
type ListRepoOpts struct {
	Archived ArchivedFilter
	Forks    ForkFilter
	Sort     string
	Order    string
	Limit    int // max total results; 0 = unlimited
	Page     int // starting page; 0 or 1 = first page
	PerPage  int // results per API request; 0 = default
}

// CreateRepoOpts holds options for creating a repository.
type CreateRepoOpts struct {
	Name          string
	Description   string
	Visibility    Visibility
	Init          bool
	DefaultBranch string
	Readme        bool
	Gitignore     string
	License       string
	Owner         string // org or group; empty = authenticated user
}

// EditRepoOpts holds options for editing a repository.
type EditRepoOpts struct {
	Description   *string
	Homepage      *string
	Visibility    Visibility
	DefaultBranch *string
	HasIssues     *bool
	HasPRs        *bool
}

// ForkRepoOpts holds options for forking a repository.
type ForkRepoOpts struct {
	Owner string // target owner/org; empty = authenticated user
	Name  string // new name; empty = keep original
}

// ListForksOpts holds options for listing forks of a repository.
type ListForksOpts struct {
	Sort    string // newest, oldest, stargazers, watchers
	Limit   int    // max total results; 0 = unlimited
	Page    int    // starting page; 0 or 1 = first page
	PerPage int    // results per API request; 0 = default
}

// SearchRepoOpts holds options for searching repositories.
type SearchRepoOpts struct {
	Query   string
	Sort    string
	Order   string
	Limit   int // max total results; 0 = unlimited
	Page    int // starting page; 0 or 1 = first page
	PerPage int // results per API request; 0 = default
}

// Label represents an issue or pull request label.
type Label struct {
	Name        string `json:"name"`
	Color       string `json:"color,omitempty"`
	Description string `json:"description,omitempty"`
}

// Milestone represents a project milestone.
type Milestone struct {
	Title       string     `json:"title"`
	Number      int        `json:"number"`
	Description string     `json:"description,omitempty"`
	State       string     `json:"state"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

// Issue holds normalized metadata about an issue.
type Issue struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"` // "open" or "closed"
	Author    User       `json:"author"`
	Assignees []User     `json:"assignees,omitempty"`
	Labels    []Label    `json:"labels,omitempty"`
	Milestone *Milestone `json:"milestone,omitempty"`
	Comments  int        `json:"comments"`
	Locked    bool       `json:"locked"`
	HTMLURL   string     `json:"html_url"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
}

// Comment holds normalized metadata about an issue or PR comment.
type Comment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	Author    User      `json:"author"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateIssueOpts holds options for creating an issue.
type CreateIssueOpts struct {
	Title     string
	Body      string
	Assignees []string
	Labels    []string
	Milestone string
}

// ListIssueOpts holds options for listing issues.
type ListIssueOpts struct {
	State    string // open, closed, all
	Labels   []string
	Assignee string
	Author   string
	Sort     string
	Order    string
	Limit    int // max total results; 0 = unlimited
	Page     int // starting page; 0 or 1 = first page
	PerPage  int // results per API request; 0 = default
}

// UpdateIssueOpts holds options for updating an issue.
type UpdateIssueOpts struct {
	Title     *string
	Body      *string
	Assignees []string
	Labels    []string
	Milestone *string
}

// PullRequest holds normalized metadata about a pull request (or merge request).
type PullRequest struct {
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	State        string     `json:"state"` // "open", "closed", or "merged"
	Draft        bool       `json:"draft"`
	Author       User       `json:"author"`
	Assignees    []User     `json:"assignees,omitempty"`
	Reviewers    []User     `json:"reviewers,omitempty"`
	Labels       []Label    `json:"labels,omitempty"`
	Milestone    *Milestone `json:"milestone,omitempty"`
	Head         string     `json:"head"` // head branch
	Base         string     `json:"base"` // base branch
	Mergeable    bool       `json:"mergeable"`
	Merged       bool       `json:"merged"`
	MergedBy     *User      `json:"merged_by,omitempty"`
	Comments     int        `json:"comments"`
	Additions    int        `json:"additions"`
	Deletions    int        `json:"deletions"`
	ChangedFiles int        `json:"changed_files"`
	HTMLURL      string     `json:"html_url"`
	DiffURL      string     `json:"diff_url,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	MergedAt     *time.Time `json:"merged_at,omitempty"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
}

// CreatePROpts holds options for creating a pull request.
type CreatePROpts struct {
	Title     string
	Body      string
	Head      string // source branch
	Base      string // target branch
	Draft     bool
	Assignees []string
	Labels    []string
	Milestone string
	Reviewers []string
}

// ListPROpts holds options for listing pull requests.
type ListPROpts struct {
	State    string // open, closed, merged, all
	Labels   []string
	Assignee string
	Author   string
	Base     string
	Head     string
	Sort     string
	Order    string
	Limit    int // max total results; 0 = unlimited
	Page     int // starting page; 0 or 1 = first page
	PerPage  int // results per API request; 0 = default
}

// UpdatePROpts holds options for updating a pull request.
type UpdatePROpts struct {
	Title     *string
	Body      *string
	Base      *string
	Assignees []string
	Labels    []string
	Milestone *string
	Reviewers []string
}

// MergePROpts holds options for merging a pull request.
type MergePROpts struct {
	Method  string // merge, squash, rebase
	Title   string // commit title
	Message string // commit message
	Delete  bool   // delete branch after merge
}

// CreateLabelOpts holds options for creating a label.
type CreateLabelOpts struct {
	Name        string
	Color       string
	Description string
}

// ListLabelOpts holds options for listing labels.
type ListLabelOpts struct {
	Limit   int // max total results; 0 = unlimited
	Page    int // starting page; 0 or 1 = first page
	PerPage int // results per API request; 0 = default
}

// UpdateLabelOpts holds options for updating a label.
type UpdateLabelOpts struct {
	Name        *string
	Color       *string
	Description *string
}

// CreateMilestoneOpts holds options for creating a milestone.
type CreateMilestoneOpts struct {
	Title       string
	Description string
	DueDate     *time.Time
}

// ListMilestoneOpts holds options for listing milestones.
type ListMilestoneOpts struct {
	State   string // open, closed, all
	Limit   int    // max total results; 0 = unlimited
	Page    int    // starting page; 0 or 1 = first page
	PerPage int    // results per API request; 0 = default
}

// UpdateMilestoneOpts holds options for updating a milestone.
type UpdateMilestoneOpts struct {
	Title       *string
	Description *string
	State       *string
	DueDate     *time.Time
}

// Release holds normalized metadata about a release.
type Release struct {
	TagName     string         `json:"tag_name"`
	Title       string         `json:"title"`
	Body        string         `json:"body,omitempty"`
	Draft       bool           `json:"draft"`
	Prerelease  bool           `json:"prerelease"`
	Target      string         `json:"target,omitempty"`
	Author      User           `json:"author"`
	Assets      []ReleaseAsset `json:"assets,omitempty"`
	TarballURL  string         `json:"tarball_url,omitempty"`
	ZipballURL  string         `json:"zipball_url,omitempty"`
	HTMLURL     string         `json:"html_url"`
	CreatedAt   time.Time      `json:"created_at"`
	PublishedAt time.Time      `json:"published_at,omitzero"`
}

// ReleaseAsset holds metadata about a file attached to a release.
type ReleaseAsset struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Size          int       `json:"size"`
	DownloadCount int       `json:"download_count"`
	DownloadURL   string    `json:"download_url"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateReleaseOpts holds options for creating a release.
type CreateReleaseOpts struct {
	TagName       string
	Target        string
	Title         string
	Body          string
	Draft         bool
	Prerelease    bool
	GenerateNotes bool
}

// ListReleaseOpts holds options for listing releases.
type ListReleaseOpts struct {
	Limit   int // max total results; 0 = unlimited
	Page    int // starting page; 0 or 1 = first page
	PerPage int // results per API request; 0 = default
}

// UpdateReleaseOpts holds options for updating a release.
type UpdateReleaseOpts struct {
	TagName    *string
	Target     *string
	Title      *string
	Body       *string
	Draft      *bool
	Prerelease *bool
}

// Branch holds normalized metadata about a git branch.
type Branch struct {
	Name      string `json:"name"`
	SHA       string `json:"sha"`
	Protected bool   `json:"protected"`
	Default   bool   `json:"default"`
}

// ListBranchOpts holds options for listing branches.
type ListBranchOpts struct {
	Limit   int // max total results; 0 = unlimited
	Page    int // starting page; 0 or 1 = first page
	PerPage int // results per API request; 0 = default
}

// CIRun holds normalized metadata about a CI pipeline or workflow run.
type CIRun struct {
	ID         int64      `json:"id"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`     // queued, running, completed, failed, success, cancelled
	Conclusion string     `json:"conclusion"` // success, failure, cancelled, skipped (GitHub-specific)
	Branch     string     `json:"branch"`
	SHA        string     `json:"sha"`
	Event      string     `json:"event,omitempty"` // push, pull_request, etc.
	Author     User       `json:"author"`
	HTMLURL    string     `json:"html_url"`
	Jobs       []CIJob    `json:"jobs,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// CIJob holds normalized metadata about a CI job.
type CIJob struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	Conclusion string     `json:"conclusion,omitempty"`
	HTMLURL    string     `json:"html_url,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// ListCIRunOpts holds options for listing CI runs.
type ListCIRunOpts struct {
	Branch   string
	Status   string
	User     string
	Workflow string
	Limit    int // max total results; 0 = unlimited
	Page     int // starting page; 0 or 1 = first page
	PerPage  int // results per API request; 0 = default
}

// TriggerCIRunOpts holds options for triggering a CI run.
type TriggerCIRunOpts struct {
	Workflow string
	Branch   string
	Inputs   map[string]string
}

// DeployKey holds normalized metadata about a deploy key.
type DeployKey struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Key       string    `json:"key"`
	ReadOnly  bool      `json:"read_only"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateDeployKeyOpts holds options for adding a deploy key.
type CreateDeployKeyOpts struct {
	Title    string
	Key      string
	ReadOnly bool
}

// ListDeployKeyOpts holds options for listing deploy keys.
type ListDeployKeyOpts struct {
	Limit   int // max total results; 0 = unlimited
	Page    int // starting page; 0 or 1 = first page
	PerPage int // results per API request; 0 = default
}

// Secret holds normalized metadata about a repository secret.
type Secret struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at,omitzero"`
	UpdatedAt time.Time `json:"updated_at,omitzero"`
}

// SetSecretOpts holds options for creating or updating a secret.
type SetSecretOpts struct {
	Name  string
	Value string
}

// ListSecretOpts holds options for listing secrets.
type ListSecretOpts struct {
	Limit   int // max total results; 0 = unlimited
	Page    int // starting page; 0 or 1 = first page
	PerPage int // results per API request; 0 = default
}

// NotificationSubjectType identifies the kind of resource a notification is about.
type NotificationSubjectType string

const (
	NotificationSubjectIssue       NotificationSubjectType = "issue"
	NotificationSubjectPullRequest NotificationSubjectType = "pull_request"
	NotificationSubjectCommit      NotificationSubjectType = "commit"
	NotificationSubjectRelease     NotificationSubjectType = "release"
	NotificationSubjectRepository  NotificationSubjectType = "repository"
	NotificationSubjectDiscussion  NotificationSubjectType = "discussion"
)

// Notification holds normalized metadata about a notification thread.
type Notification struct {
	ID          string                  `json:"id"`
	Title       string                  `json:"title"`
	SubjectType NotificationSubjectType `json:"subject_type"`
	Repo        string                  `json:"repo"`
	Unread      bool                    `json:"unread"`
	Reason      string                  `json:"reason,omitempty"`
	URL         string                  `json:"url,omitempty"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

// ListNotificationOpts holds options for listing notifications.
type ListNotificationOpts struct {
	Repo    string // filter by repo (owner/repo)
	Unread  bool   // only unread
	Limit   int    // max total results; 0 = unlimited
	Page    int    // starting page; 0 or 1 = first page
	PerPage int    // results per API request; 0 = default
}

// MarkNotificationOpts holds options for marking notifications as read.
type MarkNotificationOpts struct {
	ID   string // mark a single thread; empty = mark all
	Repo string // mark all in a repo; empty = mark all
}

// ReviewState represents the state of a pull request review.
type ReviewState string

const (
	ReviewApproved         ReviewState = "approved"
	ReviewChangesRequested ReviewState = "changes_requested"
	ReviewCommented        ReviewState = "commented"
	ReviewDismissed        ReviewState = "dismissed"
	ReviewPending          ReviewState = "pending"
)

// Review holds normalized metadata about a pull request review.
type Review struct {
	ID          int64       `json:"id"`
	State       ReviewState `json:"state"`
	Body        string      `json:"body,omitempty"`
	Author      User        `json:"author"`
	HTMLURL     string      `json:"html_url,omitempty"`
	SubmittedAt time.Time   `json:"submitted_at,omitzero"`
}

// ListReviewOpts holds options for listing reviews.
type ListReviewOpts struct {
	Limit   int // max total results; 0 = unlimited
	Page    int // starting page; 0 or 1 = first page
	PerPage int // results per API request; 0 = default
}

// SubmitReviewOpts holds options for submitting a review.
type SubmitReviewOpts struct {
	State ReviewState // approved, changes_requested, or commented
	Body  string
}

// RateLimit holds normalized rate limit information for the current token.
type RateLimit struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     time.Time `json:"reset"`
}

// FileContent holds file content retrieved from a repository.
type FileContent struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content []byte `json:"content"`
	SHA     string `json:"sha,omitempty"`
}

// FileEntry holds metadata about a directory entry in a repository.
type FileEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // file, dir, symlink
	Size int64  `json:"size"`
}

// Collaborator holds normalized metadata about a repository collaborator.
type Collaborator struct {
	Login      string `json:"login"`
	Permission string `json:"permission"` // read, write, admin
}

// AddCollaboratorOpts holds options for adding a collaborator.
type AddCollaboratorOpts struct {
	Permission string // pull, push, admin (GitHub/Gitea); guest, reporter, developer, maintainer, owner (GitLab)
}

// ListCollaboratorOpts holds options for listing collaborators.
type ListCollaboratorOpts struct {
	Limit   int // max total results; 0 = unlimited
	Page    int // starting page; 0 or 1 = first page
	PerPage int // results per API request; 0 = default
}

// Contributor holds normalized metadata about a repository contributor.
type Contributor struct {
	Login         string `json:"login"`
	Contributions int    `json:"contributions"`
	Email         string `json:"email,omitempty"`
	Name          string `json:"name,omitempty"`
}

// CommitStatus holds normalized metadata about a commit status.
type CommitStatus struct {
	State       string    `json:"state"`       // success, failure, pending, error
	Context     string    `json:"context"`     // e.g. "my-check"
	Description string    `json:"description"` // short summary
	TargetURL   string    `json:"target_url"`  // link to details
	Creator     string    `json:"creator"`     // login of who created it
	CreatedAt   time.Time `json:"created_at"`
}

// SetCommitStatusOpts holds options for creating a commit status.
type SetCommitStatusOpts struct {
	State       string // success, failure, pending, error
	Context     string // e.g. "my-check"
	Description string
	TargetURL   string
}

// Reaction holds normalized metadata about a comment reaction.
type Reaction struct {
	ID      int64  `json:"id"`
	User    string `json:"user"`
	Content string `json:"content"` // +1, -1, laugh, hooray, confused, heart, rocket, eyes
}
