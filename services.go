package forges

import (
	"context"
	"io"
	"os"
)

// RepoService provides operations on repositories.
type RepoService interface {
	Get(ctx context.Context, owner, repo string) (*Repository, error)
	List(ctx context.Context, owner string, opts ListRepoOpts) ([]Repository, error)
	Create(ctx context.Context, opts CreateRepoOpts) (*Repository, error)
	Edit(ctx context.Context, owner, repo string, opts EditRepoOpts) (*Repository, error)
	Delete(ctx context.Context, owner, repo string) error
	Fork(ctx context.Context, owner, repo string, opts ForkRepoOpts) (*Repository, error)
	ListForks(ctx context.Context, owner, repo string, opts ListForksOpts) ([]Repository, error)
	ListTags(ctx context.Context, owner, repo string) ([]Tag, error)
	ListContributors(ctx context.Context, owner, repo string) ([]Contributor, error)
	Search(ctx context.Context, opts SearchRepoOpts) ([]Repository, error)
}

// PullRequestService provides operations on pull requests (merge requests on GitLab).
type PullRequestService interface {
	Get(ctx context.Context, owner, repo string, number int) (*PullRequest, error)
	List(ctx context.Context, owner, repo string, opts ListPROpts) ([]PullRequest, error)
	Create(ctx context.Context, owner, repo string, opts CreatePROpts) (*PullRequest, error)
	Update(ctx context.Context, owner, repo string, number int, opts UpdatePROpts) (*PullRequest, error)
	Close(ctx context.Context, owner, repo string, number int) error
	Reopen(ctx context.Context, owner, repo string, number int) error
	Merge(ctx context.Context, owner, repo string, number int, opts MergePROpts) error
	Diff(ctx context.Context, owner, repo string, number int) (string, error)
	CreateComment(ctx context.Context, owner, repo string, number int, body string) (*Comment, error)
	ListComments(ctx context.Context, owner, repo string, number int) ([]Comment, error)
	ListReactions(ctx context.Context, owner, repo string, number int, commentID int64) ([]Reaction, error)
	AddReaction(ctx context.Context, owner, repo string, number int, commentID int64, reaction string) (*Reaction, error)
}

// LabelService provides operations on repository labels.
type LabelService interface {
	List(ctx context.Context, owner, repo string, opts ListLabelOpts) ([]Label, error)
	Get(ctx context.Context, owner, repo, name string) (*Label, error)
	Create(ctx context.Context, owner, repo string, opts CreateLabelOpts) (*Label, error)
	Update(ctx context.Context, owner, repo, name string, opts UpdateLabelOpts) (*Label, error)
	Delete(ctx context.Context, owner, repo, name string) error
}

// MilestoneService provides operations on repository milestones.
type MilestoneService interface {
	List(ctx context.Context, owner, repo string, opts ListMilestoneOpts) ([]Milestone, error)
	Get(ctx context.Context, owner, repo string, id int) (*Milestone, error)
	Create(ctx context.Context, owner, repo string, opts CreateMilestoneOpts) (*Milestone, error)
	Update(ctx context.Context, owner, repo string, id int, opts UpdateMilestoneOpts) (*Milestone, error)
	Close(ctx context.Context, owner, repo string, id int) error
	Reopen(ctx context.Context, owner, repo string, id int) error
	Delete(ctx context.Context, owner, repo string, id int) error
}

// ReleaseService provides operations on releases.
type ReleaseService interface {
	List(ctx context.Context, owner, repo string, opts ListReleaseOpts) ([]Release, error)
	Get(ctx context.Context, owner, repo, tag string) (*Release, error)
	GetLatest(ctx context.Context, owner, repo string) (*Release, error)
	Create(ctx context.Context, owner, repo string, opts CreateReleaseOpts) (*Release, error)
	Update(ctx context.Context, owner, repo, tag string, opts UpdateReleaseOpts) (*Release, error)
	Delete(ctx context.Context, owner, repo, tag string) error
	UploadAsset(ctx context.Context, owner, repo, tag string, file *os.File) (*ReleaseAsset, error)
	DownloadAsset(ctx context.Context, owner, repo string, assetID int64) (io.ReadCloser, error)
}

// BranchService provides operations on repository branches.
type BranchService interface {
	List(ctx context.Context, owner, repo string, opts ListBranchOpts) ([]Branch, error)
	Create(ctx context.Context, owner, repo, name, from string) (*Branch, error)
	Delete(ctx context.Context, owner, repo, name string) error
}

// CIService provides operations on CI/CD pipelines and workflow runs.
type CIService interface {
	ListRuns(ctx context.Context, owner, repo string, opts ListCIRunOpts) ([]CIRun, error)
	GetRun(ctx context.Context, owner, repo string, runID int64) (*CIRun, error)
	TriggerRun(ctx context.Context, owner, repo string, opts TriggerCIRunOpts) error
	CancelRun(ctx context.Context, owner, repo string, runID int64) error
	RetryRun(ctx context.Context, owner, repo string, runID int64) error
	GetJobLog(ctx context.Context, owner, repo string, jobID int64) (io.ReadCloser, error)
}

// DeployKeyService provides operations on repository deploy keys.
type DeployKeyService interface {
	List(ctx context.Context, owner, repo string, opts ListDeployKeyOpts) ([]DeployKey, error)
	Get(ctx context.Context, owner, repo string, id int64) (*DeployKey, error)
	Create(ctx context.Context, owner, repo string, opts CreateDeployKeyOpts) (*DeployKey, error)
	Delete(ctx context.Context, owner, repo string, id int64) error
}

// SecretService provides operations on repository secrets.
type SecretService interface {
	List(ctx context.Context, owner, repo string, opts ListSecretOpts) ([]Secret, error)
	Set(ctx context.Context, owner, repo string, opts SetSecretOpts) error
	Delete(ctx context.Context, owner, repo, name string) error
}

// NotificationService provides operations on user notifications.
type NotificationService interface {
	List(ctx context.Context, opts ListNotificationOpts) ([]Notification, error)
	MarkRead(ctx context.Context, opts MarkNotificationOpts) error
	Get(ctx context.Context, id string) (*Notification, error)
}

// ReviewService provides operations on pull request reviews.
type ReviewService interface {
	List(ctx context.Context, owner, repo string, number int, opts ListReviewOpts) ([]Review, error)
	Submit(ctx context.Context, owner, repo string, number int, opts SubmitReviewOpts) (*Review, error)
	RequestReviewers(ctx context.Context, owner, repo string, number int, users []string) error
	RemoveReviewers(ctx context.Context, owner, repo string, number int, users []string) error
}

// FileService provides operations on file content within a repository.
type FileService interface {
	Get(ctx context.Context, owner, repo, path, ref string) (*FileContent, error)
	List(ctx context.Context, owner, repo, path, ref string) ([]FileEntry, error)
}

// CollaboratorService provides operations on repository collaborators.
type CollaboratorService interface {
	List(ctx context.Context, owner, repo string, opts ListCollaboratorOpts) ([]Collaborator, error)
	Add(ctx context.Context, owner, repo, username string, opts AddCollaboratorOpts) error
	Remove(ctx context.Context, owner, repo, username string) error
}

// IssueService provides operations on issues.
type IssueService interface {
	Get(ctx context.Context, owner, repo string, number int) (*Issue, error)
	List(ctx context.Context, owner, repo string, opts ListIssueOpts) ([]Issue, error)
	Create(ctx context.Context, owner, repo string, opts CreateIssueOpts) (*Issue, error)
	Update(ctx context.Context, owner, repo string, number int, opts UpdateIssueOpts) (*Issue, error)
	Close(ctx context.Context, owner, repo string, number int) error
	Reopen(ctx context.Context, owner, repo string, number int) error
	Delete(ctx context.Context, owner, repo string, number int) error
	CreateComment(ctx context.Context, owner, repo string, number int, body string) (*Comment, error)
	ListComments(ctx context.Context, owner, repo string, number int) ([]Comment, error)
	ListReactions(ctx context.Context, owner, repo string, number int, commentID int64) ([]Reaction, error)
	AddReaction(ctx context.Context, owner, repo string, number int, commentID int64, reaction string) (*Reaction, error)
}
