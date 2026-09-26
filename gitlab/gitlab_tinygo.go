//go:build tinygo

package gitlab

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	forge "github.com/git-pkgs/forge"
)

var errTinyGo = fmt.Errorf("gitlab: API operations are unavailable under TinyGo: %w", forge.ErrNotSupported)

type gitLabForge struct{ apiBaseURL string }

// New creates a GitLab backend whose API operations return forge.ErrNotSupported.
func New(baseURL, _ string, _ *http.Client) forge.Forge {
	return &gitLabForge{apiBaseURL: strings.TrimRight(baseURL, "/") + "/api/v4"}
}

func (f *gitLabForge) APIBaseURL() string { return f.apiBaseURL }

func (f *gitLabForge) GetRateLimit(context.Context) (*forge.RateLimit, error) {
	return nil, errTinyGo
}

type gitLabBranchService struct{}

func (f *gitLabForge) Branches() forge.BranchService { return &gitLabBranchService{} }

func (s *gitLabBranchService) List(ctx context.Context, owner, repo string, opts forge.ListBranchOpts) ([]forge.Branch, error) {
	return nil, errTinyGo
}

func (s *gitLabBranchService) Create(ctx context.Context, owner, repo, name, from string) (*forge.Branch, error) {
	return nil, errTinyGo
}

func (s *gitLabBranchService) Delete(ctx context.Context, owner, repo, name string) error {
	return errTinyGo
}

type gitLabCIService struct{}

func (f *gitLabForge) CI() forge.CIService { return &gitLabCIService{} }

func (s *gitLabCIService) ListRuns(ctx context.Context, owner, repo string, opts forge.ListCIRunOpts) ([]forge.CIRun, error) {
	return nil, errTinyGo
}

func (s *gitLabCIService) GetRun(ctx context.Context, owner, repo string, runID int64) (*forge.CIRun, error) {
	return nil, errTinyGo
}

func (s *gitLabCIService) TriggerRun(ctx context.Context, owner, repo string, opts forge.TriggerCIRunOpts) error {
	return errTinyGo
}

func (s *gitLabCIService) CancelRun(ctx context.Context, owner, repo string, runID int64) error {
	return errTinyGo
}

func (s *gitLabCIService) RetryRun(ctx context.Context, owner, repo string, runID int64) error {
	return errTinyGo
}

func (s *gitLabCIService) GetJobLog(ctx context.Context, owner, repo string, jobID int64) (io.ReadCloser, error) {
	return nil, errTinyGo
}

type gitLabCollaboratorService struct{}

func (f *gitLabForge) Collaborators() forge.CollaboratorService { return &gitLabCollaboratorService{} }

func (s *gitLabCollaboratorService) List(ctx context.Context, owner, repo string, opts forge.ListCollaboratorOpts) ([]forge.Collaborator, error) {
	return nil, errTinyGo
}

func (s *gitLabCollaboratorService) Add(ctx context.Context, owner, repo, username string, opts forge.AddCollaboratorOpts) error {
	return errTinyGo
}

func (s *gitLabCollaboratorService) Remove(ctx context.Context, owner, repo, username string) error {
	return errTinyGo
}

type gitLabCommitStatusService struct{}

func (f *gitLabForge) CommitStatuses() forge.CommitStatusService { return &gitLabCommitStatusService{} }

func (s *gitLabCommitStatusService) List(ctx context.Context, owner, repo, sha string) ([]forge.CommitStatus, error) {
	return nil, errTinyGo
}

func (s *gitLabCommitStatusService) Set(ctx context.Context, owner, repo, sha string, opts forge.SetCommitStatusOpts) (*forge.CommitStatus, error) {
	return nil, errTinyGo
}

type gitLabCommitService struct{}

func (f *gitLabForge) Commits() forge.CommitService { return &gitLabCommitService{} }

func (s *gitLabCommitService) ResolveCommit(ctx context.Context, owner, repo, ref string) (string, error) {
	return "", errTinyGo
}

type gitLabDeployKeyService struct{}

func (f *gitLabForge) DeployKeys() forge.DeployKeyService { return &gitLabDeployKeyService{} }

func (s *gitLabDeployKeyService) List(ctx context.Context, owner, repo string, opts forge.ListDeployKeyOpts) ([]forge.DeployKey, error) {
	return nil, errTinyGo
}

func (s *gitLabDeployKeyService) Get(ctx context.Context, owner, repo string, id int64) (*forge.DeployKey, error) {
	return nil, errTinyGo
}

func (s *gitLabDeployKeyService) Create(ctx context.Context, owner, repo string, opts forge.CreateDeployKeyOpts) (*forge.DeployKey, error) {
	return nil, errTinyGo
}

func (s *gitLabDeployKeyService) Delete(ctx context.Context, owner, repo string, id int64) error {
	return errTinyGo
}

type gitLabFileService struct{}

func (f *gitLabForge) Files() forge.FileService { return &gitLabFileService{} }

func (s *gitLabFileService) Get(ctx context.Context, owner, repo, path, ref string) (*forge.FileContent, error) {
	return nil, errTinyGo
}

func (s *gitLabFileService) List(ctx context.Context, owner, repo, path, ref string) ([]forge.FileEntry, error) {
	return nil, errTinyGo
}

type gitLabRepoService struct{}

func (f *gitLabForge) Repos() forge.RepoService { return &gitLabRepoService{} }

func (s *gitLabRepoService) Get(ctx context.Context, owner, repo string) (*forge.Repository, error) {
	return nil, errTinyGo
}

func (s *gitLabRepoService) List(ctx context.Context, owner string, opts forge.ListRepoOpts) ([]forge.Repository, error) {
	return nil, errTinyGo
}

func (s *gitLabRepoService) Create(ctx context.Context, opts forge.CreateRepoOpts) (*forge.Repository, error) {
	return nil, errTinyGo
}

func (s *gitLabRepoService) Edit(ctx context.Context, owner, repo string, opts forge.EditRepoOpts) (*forge.Repository, error) {
	return nil, errTinyGo
}

func (s *gitLabRepoService) Delete(ctx context.Context, owner, repo string) error { return errTinyGo }

func (s *gitLabRepoService) Fork(ctx context.Context, owner, repo string, opts forge.ForkRepoOpts) (*forge.Repository, error) {
	return nil, errTinyGo
}

func (s *gitLabRepoService) ListForks(ctx context.Context, owner, repo string, opts forge.ListForksOpts) ([]forge.Repository, error) {
	return nil, errTinyGo
}

func (s *gitLabRepoService) ListTags(ctx context.Context, owner, repo string) ([]forge.Tag, error) {
	return nil, errTinyGo
}

func (s *gitLabRepoService) ListContributors(ctx context.Context, owner, repo string) ([]forge.Contributor, error) {
	return nil, errTinyGo
}

func (s *gitLabRepoService) Search(ctx context.Context, opts forge.SearchRepoOpts) ([]forge.Repository, error) {
	return nil, errTinyGo
}

type gitLabIssueService struct{}

func (f *gitLabForge) Issues() forge.IssueService { return &gitLabIssueService{} }

func (s *gitLabIssueService) Get(ctx context.Context, owner, repo string, number int) (*forge.Issue, error) {
	return nil, errTinyGo
}

func (s *gitLabIssueService) List(ctx context.Context, owner, repo string, opts forge.ListIssueOpts) ([]forge.Issue, error) {
	return nil, errTinyGo
}

func (s *gitLabIssueService) Create(ctx context.Context, owner, repo string, opts forge.CreateIssueOpts) (*forge.Issue, error) {
	return nil, errTinyGo
}

func (s *gitLabIssueService) Update(ctx context.Context, owner, repo string, number int, opts forge.UpdateIssueOpts) (*forge.Issue, error) {
	return nil, errTinyGo
}

func (s *gitLabIssueService) Close(ctx context.Context, owner, repo string, number int) error {
	return errTinyGo
}

func (s *gitLabIssueService) Reopen(ctx context.Context, owner, repo string, number int) error {
	return errTinyGo
}

func (s *gitLabIssueService) Delete(ctx context.Context, owner, repo string, number int) error {
	return errTinyGo
}

func (s *gitLabIssueService) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*forge.Comment, error) {
	return nil, errTinyGo
}

func (s *gitLabIssueService) ListComments(ctx context.Context, owner, repo string, number int) ([]forge.Comment, error) {
	return nil, errTinyGo
}

func (s *gitLabIssueService) ListReactions(ctx context.Context, owner, repo string, number int, commentID int64) ([]forge.Reaction, error) {
	return nil, errTinyGo
}

func (s *gitLabIssueService) AddReaction(ctx context.Context, owner, repo string, number int, commentID int64, reaction string) (*forge.Reaction, error) {
	return nil, errTinyGo
}

type gitLabLabelService struct{}

func (f *gitLabForge) Labels() forge.LabelService { return &gitLabLabelService{} }

func (s *gitLabLabelService) List(ctx context.Context, owner, repo string, opts forge.ListLabelOpts) ([]forge.Label, error) {
	return nil, errTinyGo
}

func (s *gitLabLabelService) Get(ctx context.Context, owner, repo, name string) (*forge.Label, error) {
	return nil, errTinyGo
}

func (s *gitLabLabelService) Create(ctx context.Context, owner, repo string, opts forge.CreateLabelOpts) (*forge.Label, error) {
	return nil, errTinyGo
}

func (s *gitLabLabelService) Update(ctx context.Context, owner, repo, name string, opts forge.UpdateLabelOpts) (*forge.Label, error) {
	return nil, errTinyGo
}

func (s *gitLabLabelService) Delete(ctx context.Context, owner, repo, name string) error {
	return errTinyGo
}

type gitLabMilestoneService struct{}

func (f *gitLabForge) Milestones() forge.MilestoneService { return &gitLabMilestoneService{} }

func (s *gitLabMilestoneService) List(ctx context.Context, owner, repo string, opts forge.ListMilestoneOpts) ([]forge.Milestone, error) {
	return nil, errTinyGo
}

func (s *gitLabMilestoneService) Get(ctx context.Context, owner, repo string, id int) (*forge.Milestone, error) {
	return nil, errTinyGo
}

func (s *gitLabMilestoneService) Create(ctx context.Context, owner, repo string, opts forge.CreateMilestoneOpts) (*forge.Milestone, error) {
	return nil, errTinyGo
}

func (s *gitLabMilestoneService) Update(ctx context.Context, owner, repo string, id int, opts forge.UpdateMilestoneOpts) (*forge.Milestone, error) {
	return nil, errTinyGo
}

func (s *gitLabMilestoneService) Close(ctx context.Context, owner, repo string, id int) error {
	return errTinyGo
}

func (s *gitLabMilestoneService) Reopen(ctx context.Context, owner, repo string, id int) error {
	return errTinyGo
}

func (s *gitLabMilestoneService) Delete(ctx context.Context, owner, repo string, id int) error {
	return errTinyGo
}

type gitLabNotificationService struct{}

func (f *gitLabForge) Notifications() forge.NotificationService { return &gitLabNotificationService{} }

func (s *gitLabNotificationService) List(ctx context.Context, opts forge.ListNotificationOpts) ([]forge.Notification, error) {
	return nil, errTinyGo
}

func (s *gitLabNotificationService) MarkRead(ctx context.Context, opts forge.MarkNotificationOpts) error {
	return errTinyGo
}

func (s *gitLabNotificationService) Get(ctx context.Context, id string) (*forge.Notification, error) {
	return nil, errTinyGo
}

type gitLabPRService struct{}

func (f *gitLabForge) PullRequests() forge.PullRequestService { return &gitLabPRService{} }

func (s *gitLabPRService) Get(ctx context.Context, owner, repo string, number int) (*forge.PullRequest, error) {
	return nil, errTinyGo
}

func (s *gitLabPRService) List(ctx context.Context, owner, repo string, opts forge.ListPROpts) ([]forge.PullRequest, error) {
	return nil, errTinyGo
}

func (s *gitLabPRService) Create(ctx context.Context, owner, repo string, opts forge.CreatePROpts) (*forge.PullRequest, error) {
	return nil, errTinyGo
}

func (s *gitLabPRService) Update(ctx context.Context, owner, repo string, number int, opts forge.UpdatePROpts) (*forge.PullRequest, error) {
	return nil, errTinyGo
}

func (s *gitLabPRService) Close(ctx context.Context, owner, repo string, number int) error {
	return errTinyGo
}

func (s *gitLabPRService) Reopen(ctx context.Context, owner, repo string, number int) error {
	return errTinyGo
}

func (s *gitLabPRService) Merge(ctx context.Context, owner, repo string, number int, opts forge.MergePROpts) error {
	return errTinyGo
}

func (s *gitLabPRService) Diff(ctx context.Context, owner, repo string, number int) (string, error) {
	return "", errTinyGo
}

func (s *gitLabPRService) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*forge.Comment, error) {
	return nil, errTinyGo
}

func (s *gitLabPRService) ListComments(ctx context.Context, owner, repo string, number int) ([]forge.Comment, error) {
	return nil, errTinyGo
}

func (s *gitLabPRService) ListReactions(ctx context.Context, owner, repo string, number int, commentID int64) ([]forge.Reaction, error) {
	return nil, errTinyGo
}

func (s *gitLabPRService) AddReaction(ctx context.Context, owner, repo string, number int, commentID int64, reaction string) (*forge.Reaction, error) {
	return nil, errTinyGo
}

type gitLabReleaseService struct{}

func (f *gitLabForge) Releases() forge.ReleaseService { return &gitLabReleaseService{} }

func (s *gitLabReleaseService) List(ctx context.Context, owner, repo string, opts forge.ListReleaseOpts) ([]forge.Release, error) {
	return nil, errTinyGo
}

func (s *gitLabReleaseService) Get(ctx context.Context, owner, repo, tag string) (*forge.Release, error) {
	return nil, errTinyGo
}

func (s *gitLabReleaseService) GetLatest(ctx context.Context, owner, repo string) (*forge.Release, error) {
	return nil, errTinyGo
}

func (s *gitLabReleaseService) Create(ctx context.Context, owner, repo string, opts forge.CreateReleaseOpts) (*forge.Release, error) {
	return nil, errTinyGo
}

func (s *gitLabReleaseService) Update(ctx context.Context, owner, repo, tag string, opts forge.UpdateReleaseOpts) (*forge.Release, error) {
	return nil, errTinyGo
}

func (s *gitLabReleaseService) Delete(ctx context.Context, owner, repo, tag string) error {
	return errTinyGo
}

func (s *gitLabReleaseService) UploadAsset(ctx context.Context, owner, repo, tag string, file *os.File) (*forge.ReleaseAsset, error) {
	return nil, errTinyGo
}

func (s *gitLabReleaseService) DownloadAsset(ctx context.Context, owner, repo string, assetID int64) (io.ReadCloser, error) {
	return nil, errTinyGo
}

type gitLabReviewService struct{}

func (f *gitLabForge) Reviews() forge.ReviewService { return &gitLabReviewService{} }

func (s *gitLabReviewService) List(ctx context.Context, owner, repo string, number int, opts forge.ListReviewOpts) ([]forge.Review, error) {
	return nil, errTinyGo
}

func (s *gitLabReviewService) Submit(ctx context.Context, owner, repo string, number int, opts forge.SubmitReviewOpts) (*forge.Review, error) {
	return nil, errTinyGo
}

func (s *gitLabReviewService) RequestReviewers(ctx context.Context, owner, repo string, number int, users []string) error {
	return errTinyGo
}

func (s *gitLabReviewService) RemoveReviewers(ctx context.Context, owner, repo string, number int, users []string) error {
	return errTinyGo
}

type gitLabSecretService struct{}

func (f *gitLabForge) Secrets() forge.SecretService { return &gitLabSecretService{} }

func (s *gitLabSecretService) List(ctx context.Context, owner, repo string, opts forge.ListSecretOpts) ([]forge.Secret, error) {
	return nil, errTinyGo
}

func (s *gitLabSecretService) Set(ctx context.Context, owner, repo string, opts forge.SetSecretOpts) error {
	return errTinyGo
}

func (s *gitLabSecretService) Delete(ctx context.Context, owner, repo, name string) error {
	return errTinyGo
}
