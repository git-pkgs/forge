package tangled

import (
	"context"
	"io"
	"os"

	forges "github.com/git-pkgs/forge"
)

type unsupportedIssueService struct{}

func (unsupportedIssueService) Get(context.Context, string, string, int) (*forges.Issue, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedIssueService) List(context.Context, string, string, forges.ListIssueOpts) ([]forges.Issue, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedIssueService) Create(context.Context, string, string, forges.CreateIssueOpts) (*forges.Issue, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedIssueService) Update(context.Context, string, string, int, forges.UpdateIssueOpts) (*forges.Issue, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedIssueService) Close(context.Context, string, string, int) error {
	return forges.ErrNotSupported
}
func (unsupportedIssueService) Reopen(context.Context, string, string, int) error {
	return forges.ErrNotSupported
}
func (unsupportedIssueService) Delete(context.Context, string, string, int) error {
	return forges.ErrNotSupported
}
func (unsupportedIssueService) CreateComment(context.Context, string, string, int, string) (*forges.Comment, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedIssueService) ListComments(context.Context, string, string, int) ([]forges.Comment, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedIssueService) ListReactions(context.Context, string, string, int, int64) ([]forges.Reaction, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedIssueService) AddReaction(context.Context, string, string, int, int64, string) (*forges.Reaction, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedIssueService) ListURL(repoHTMLURL string) string { return repoHTMLURL + "/issues" }

type unsupportedPRService struct{}

func (unsupportedPRService) Get(context.Context, string, string, int) (*forges.PullRequest, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedPRService) List(context.Context, string, string, forges.ListPROpts) ([]forges.PullRequest, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedPRService) Create(context.Context, string, string, forges.CreatePROpts) (*forges.PullRequest, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedPRService) Update(context.Context, string, string, int, forges.UpdatePROpts) (*forges.PullRequest, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedPRService) Close(context.Context, string, string, int) error {
	return forges.ErrNotSupported
}
func (unsupportedPRService) Reopen(context.Context, string, string, int) error {
	return forges.ErrNotSupported
}
func (unsupportedPRService) Merge(context.Context, string, string, int, forges.MergePROpts) error {
	return forges.ErrNotSupported
}
func (unsupportedPRService) Diff(context.Context, string, string, int) (string, error) {
	return "", forges.ErrNotSupported
}
func (unsupportedPRService) CreateComment(context.Context, string, string, int, string) (*forges.Comment, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedPRService) ListComments(context.Context, string, string, int) ([]forges.Comment, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedPRService) ListReactions(context.Context, string, string, int, int64) ([]forges.Reaction, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedPRService) AddReaction(context.Context, string, string, int, int64, string) (*forges.Reaction, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedPRService) ListURL(repoHTMLURL string) string { return repoHTMLURL + "/pulls" }

type unsupportedLabelService struct{}

func (unsupportedLabelService) List(context.Context, string, string, forges.ListLabelOpts) ([]forges.Label, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedLabelService) Get(context.Context, string, string, string) (*forges.Label, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedLabelService) Create(context.Context, string, string, forges.CreateLabelOpts) (*forges.Label, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedLabelService) Update(context.Context, string, string, string, forges.UpdateLabelOpts) (*forges.Label, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedLabelService) Delete(context.Context, string, string, string) error {
	return forges.ErrNotSupported
}
func (unsupportedLabelService) ListURL(repoHTMLURL string) string { return repoHTMLURL + "/labels" }

type unsupportedMilestoneService struct{}

func (unsupportedMilestoneService) List(context.Context, string, string, forges.ListMilestoneOpts) ([]forges.Milestone, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedMilestoneService) Get(context.Context, string, string, int) (*forges.Milestone, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedMilestoneService) Create(context.Context, string, string, forges.CreateMilestoneOpts) (*forges.Milestone, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedMilestoneService) Update(context.Context, string, string, int, forges.UpdateMilestoneOpts) (*forges.Milestone, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedMilestoneService) Close(context.Context, string, string, int) error {
	return forges.ErrNotSupported
}
func (unsupportedMilestoneService) Reopen(context.Context, string, string, int) error {
	return forges.ErrNotSupported
}
func (unsupportedMilestoneService) Delete(context.Context, string, string, int) error {
	return forges.ErrNotSupported
}

type unsupportedReleaseService struct{}

func (unsupportedReleaseService) List(context.Context, string, string, forges.ListReleaseOpts) ([]forges.Release, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedReleaseService) Get(context.Context, string, string, string) (*forges.Release, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedReleaseService) GetLatest(context.Context, string, string) (*forges.Release, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedReleaseService) Create(context.Context, string, string, forges.CreateReleaseOpts) (*forges.Release, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedReleaseService) Update(context.Context, string, string, string, forges.UpdateReleaseOpts) (*forges.Release, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedReleaseService) Delete(context.Context, string, string, string) error {
	return forges.ErrNotSupported
}
func (unsupportedReleaseService) UploadAsset(context.Context, string, string, string, *os.File) (*forges.ReleaseAsset, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedReleaseService) DownloadAsset(context.Context, string, string, int64) (io.ReadCloser, error) {
	return nil, forges.ErrNotSupported
}

type unsupportedCIService struct{}

func (unsupportedCIService) ListRuns(context.Context, string, string, forges.ListCIRunOpts) ([]forges.CIRun, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedCIService) GetRun(context.Context, string, string, int64) (*forges.CIRun, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedCIService) TriggerRun(context.Context, string, string, forges.TriggerCIRunOpts) error {
	return forges.ErrNotSupported
}
func (unsupportedCIService) CancelRun(context.Context, string, string, int64) error {
	return forges.ErrNotSupported
}
func (unsupportedCIService) RetryRun(context.Context, string, string, int64) error {
	return forges.ErrNotSupported
}
func (unsupportedCIService) GetJobLog(context.Context, string, string, int64) (io.ReadCloser, error) {
	return nil, forges.ErrNotSupported
}

type unsupportedDeployKeyService struct{}

func (unsupportedDeployKeyService) List(context.Context, string, string, forges.ListDeployKeyOpts) ([]forges.DeployKey, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedDeployKeyService) Get(context.Context, string, string, int64) (*forges.DeployKey, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedDeployKeyService) Create(context.Context, string, string, forges.CreateDeployKeyOpts) (*forges.DeployKey, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedDeployKeyService) Delete(context.Context, string, string, int64) error {
	return forges.ErrNotSupported
}

type unsupportedSecretService struct{}

func (unsupportedSecretService) List(context.Context, string, string, forges.ListSecretOpts) ([]forges.Secret, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedSecretService) Set(context.Context, string, string, forges.SetSecretOpts) error {
	return forges.ErrNotSupported
}
func (unsupportedSecretService) Delete(context.Context, string, string, string) error {
	return forges.ErrNotSupported
}

type unsupportedNotificationService struct{}

func (unsupportedNotificationService) List(context.Context, forges.ListNotificationOpts) ([]forges.Notification, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedNotificationService) MarkRead(context.Context, forges.MarkNotificationOpts) error {
	return forges.ErrNotSupported
}
func (unsupportedNotificationService) Get(context.Context, string) (*forges.Notification, error) {
	return nil, forges.ErrNotSupported
}

type unsupportedReviewService struct{}

func (unsupportedReviewService) List(context.Context, string, string, int, forges.ListReviewOpts) ([]forges.Review, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedReviewService) Submit(context.Context, string, string, int, forges.SubmitReviewOpts) (*forges.Review, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedReviewService) RequestReviewers(context.Context, string, string, int, []string) error {
	return forges.ErrNotSupported
}
func (unsupportedReviewService) RemoveReviewers(context.Context, string, string, int, []string) error {
	return forges.ErrNotSupported
}

type unsupportedCollaboratorService struct{}

func (unsupportedCollaboratorService) List(context.Context, string, string, forges.ListCollaboratorOpts) ([]forges.Collaborator, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedCollaboratorService) Add(context.Context, string, string, string, forges.AddCollaboratorOpts) error {
	return forges.ErrNotSupported
}
func (unsupportedCollaboratorService) Remove(context.Context, string, string, string) error {
	return forges.ErrNotSupported
}

type unsupportedCommitStatusService struct{}

func (unsupportedCommitStatusService) List(context.Context, string, string, string) ([]forges.CommitStatus, error) {
	return nil, forges.ErrNotSupported
}
func (unsupportedCommitStatusService) Set(context.Context, string, string, string, forges.SetCommitStatusOpts) (*forges.CommitStatus, error) {
	return nil, forges.ErrNotSupported
}
