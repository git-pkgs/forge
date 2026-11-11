package gerrit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"

	forge "github.com/git-pkgs/forge"
)

type unsupportedIssueService struct{}
type unsupportedLabelService struct{}
type unsupportedMilestoneService struct{}
type unsupportedReleaseService struct{}
type unsupportedCIService struct{}
type unsupportedDeployKeyService struct{}
type unsupportedWebhookService struct{}
type unsupportedSecretService struct{}
type unsupportedNotificationService struct{}
type unsupportedCollaboratorService struct{}
type unsupportedCommitStatusService struct{}
type unsupportedCheckRunService struct{}
type unsupportedCommitService struct{}
type unsupportedAttestationService struct{}

func (f *gerritForge) Issues() forge.IssueService         { return &unsupportedIssueService{} }
func (f *gerritForge) Labels() forge.LabelService         { return &unsupportedLabelService{} }
func (f *gerritForge) Milestones() forge.MilestoneService { return &unsupportedMilestoneService{} }
func (f *gerritForge) Releases() forge.ReleaseService     { return &unsupportedReleaseService{} }
func (f *gerritForge) CI() forge.CIService                { return &unsupportedCIService{} }
func (f *gerritForge) DeployKeys() forge.DeployKeyService { return &unsupportedDeployKeyService{} }
func (f *gerritForge) Webhooks() forge.WebhookService     { return &unsupportedWebhookService{} }
func (f *gerritForge) Secrets() forge.SecretService       { return &unsupportedSecretService{} }
func (f *gerritForge) Notifications() forge.NotificationService {
	return &unsupportedNotificationService{}
}
func (f *gerritForge) Collaborators() forge.CollaboratorService {
	return &unsupportedCollaboratorService{}
}
func (f *gerritForge) CommitStatuses() forge.CommitStatusService {
	return &unsupportedCommitStatusService{}
}
func (f *gerritForge) CheckRuns() forge.CheckRunService {
	return &unsupportedCheckRunService{}
}
func (f *gerritForge) Commits() forge.CommitService { return &unsupportedCommitService{} }
func (f *gerritForge) Attestations() forge.AttestationService {
	return &unsupportedAttestationService{}
}
func (f *gerritForge) GetRateLimit(context.Context) (*forge.RateLimit, error) {
	return nil, forge.ErrNotSupported
}

func (s *unsupportedIssueService) Get(context.Context, string, string, int) (*forge.Issue, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedIssueService) List(context.Context, string, string, forge.ListIssueOpts) ([]forge.Issue, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedIssueService) Create(context.Context, string, string, forge.CreateIssueOpts) (*forge.Issue, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedIssueService) Update(context.Context, string, string, int, forge.UpdateIssueOpts) (*forge.Issue, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedIssueService) Close(context.Context, string, string, int) error {
	return forge.ErrNotSupported
}
func (s *unsupportedIssueService) Reopen(context.Context, string, string, int) error {
	return forge.ErrNotSupported
}
func (s *unsupportedIssueService) Delete(context.Context, string, string, int) error {
	return forge.ErrNotSupported
}
func (s *unsupportedIssueService) CreateComment(context.Context, string, string, int, string) (*forge.Comment, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedIssueService) ListComments(context.Context, string, string, int) ([]forge.Comment, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedIssueService) ListReactions(context.Context, string, string, int, int64) ([]forge.Reaction, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedIssueService) AddReaction(context.Context, string, string, int, int64, string) (*forge.Reaction, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedIssueService) ListURL(repoHTMLURL string) string { return repoHTMLURL }

func (s *unsupportedLabelService) List(context.Context, string, string, forge.ListLabelOpts) ([]forge.Label, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedLabelService) Get(context.Context, string, string, string) (*forge.Label, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedLabelService) Create(context.Context, string, string, forge.CreateLabelOpts) (*forge.Label, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedLabelService) Update(context.Context, string, string, string, forge.UpdateLabelOpts) (*forge.Label, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedLabelService) Delete(context.Context, string, string, string) error {
	return forge.ErrNotSupported
}
func (s *unsupportedLabelService) ListURL(repoHTMLURL string) string { return repoHTMLURL }

func (s *unsupportedMilestoneService) List(context.Context, string, string, forge.ListMilestoneOpts) ([]forge.Milestone, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedMilestoneService) Get(context.Context, string, string, int) (*forge.Milestone, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedMilestoneService) Create(context.Context, string, string, forge.CreateMilestoneOpts) (*forge.Milestone, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedMilestoneService) Update(context.Context, string, string, int, forge.UpdateMilestoneOpts) (*forge.Milestone, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedMilestoneService) Close(context.Context, string, string, int) error {
	return forge.ErrNotSupported
}
func (s *unsupportedMilestoneService) Reopen(context.Context, string, string, int) error {
	return forge.ErrNotSupported
}
func (s *unsupportedMilestoneService) Delete(context.Context, string, string, int) error {
	return forge.ErrNotSupported
}

func (s *unsupportedReleaseService) List(context.Context, string, string, forge.ListReleaseOpts) ([]forge.Release, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedReleaseService) Get(context.Context, string, string, string) (*forge.Release, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedReleaseService) GetLatest(context.Context, string, string) (*forge.Release, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedReleaseService) Create(context.Context, string, string, forge.CreateReleaseOpts) (*forge.Release, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedReleaseService) Update(context.Context, string, string, string, forge.UpdateReleaseOpts) (*forge.Release, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedReleaseService) Delete(context.Context, string, string, string) error {
	return forge.ErrNotSupported
}
func (s *unsupportedReleaseService) UploadAsset(context.Context, string, string, string, *os.File) (*forge.ReleaseAsset, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedReleaseService) DownloadAsset(context.Context, string, string, int64) (io.ReadCloser, error) {
	return nil, forge.ErrNotSupported
}

func (s *unsupportedCIService) ListRuns(context.Context, string, string, forge.ListCIRunOpts) ([]forge.CIRun, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedCIService) GetRun(context.Context, string, string, int64) (*forge.CIRun, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedCIService) TriggerRun(context.Context, string, string, forge.TriggerCIRunOpts) error {
	return forge.ErrNotSupported
}
func (s *unsupportedCIService) CancelRun(context.Context, string, string, int64) error {
	return forge.ErrNotSupported
}
func (s *unsupportedCIService) RetryRun(context.Context, string, string, int64) error {
	return forge.ErrNotSupported
}
func (s *unsupportedCIService) GetJobLog(context.Context, string, string, int64) (io.ReadCloser, error) {
	return nil, forge.ErrNotSupported
}

func (s *unsupportedDeployKeyService) List(context.Context, string, string, forge.ListDeployKeyOpts) ([]forge.DeployKey, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedDeployKeyService) Get(context.Context, string, string, int64) (*forge.DeployKey, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedDeployKeyService) Create(context.Context, string, string, forge.CreateDeployKeyOpts) (*forge.DeployKey, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedDeployKeyService) Delete(context.Context, string, string, int64) error {
	return forge.ErrNotSupported
}

func (s *unsupportedWebhookService) List(context.Context, string, string, forge.ListWebhookOpts) ([]forge.Webhook, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedWebhookService) Get(context.Context, string, string, int64) (*forge.Webhook, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedWebhookService) Create(context.Context, string, string, forge.CreateWebhookOpts) (*forge.Webhook, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedWebhookService) Delete(context.Context, string, string, int64) error {
	return forge.ErrNotSupported
}
func (s *unsupportedWebhookService) ParseEvent(*http.Request, string) (*forge.WebhookEvent, error) {
	return nil, forge.ErrNotSupported
}

func (s *unsupportedSecretService) List(context.Context, string, string, forge.ListSecretOpts) ([]forge.Secret, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedSecretService) Set(context.Context, string, string, forge.SetSecretOpts) error {
	return forge.ErrNotSupported
}
func (s *unsupportedSecretService) Delete(context.Context, string, string, string) error {
	return forge.ErrNotSupported
}

func (s *unsupportedNotificationService) List(context.Context, forge.ListNotificationOpts) ([]forge.Notification, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedNotificationService) MarkRead(context.Context, forge.MarkNotificationOpts) error {
	return forge.ErrNotSupported
}
func (s *unsupportedNotificationService) Get(context.Context, string) (*forge.Notification, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedNotificationService) GetRepoSubscription(context.Context, string, string) (*forge.Subscription, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedNotificationService) SetRepoSubscription(context.Context, string, string, forge.SetSubscriptionOpts) (*forge.Subscription, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedNotificationService) DeleteRepoSubscription(context.Context, string, string) error {
	return forge.ErrNotSupported
}
func (s *unsupportedNotificationService) GetThreadSubscription(context.Context, string) (*forge.Subscription, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedNotificationService) SetThreadSubscription(context.Context, string, forge.SetSubscriptionOpts) (*forge.Subscription, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedNotificationService) DeleteThreadSubscription(context.Context, string) error {
	return forge.ErrNotSupported
}

func (s *unsupportedCollaboratorService) List(context.Context, string, string, forge.ListCollaboratorOpts) ([]forge.Collaborator, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedCollaboratorService) Add(context.Context, string, string, string, forge.AddCollaboratorOpts) error {
	return forge.ErrNotSupported
}
func (s *unsupportedCollaboratorService) Remove(context.Context, string, string, string) error {
	return forge.ErrNotSupported
}

func (s *unsupportedCommitStatusService) List(context.Context, string, string, string) ([]forge.CommitStatus, error) {
	return nil, forge.ErrNotSupported
}
func (s *unsupportedCommitStatusService) Set(context.Context, string, string, string, forge.SetCommitStatusOpts) (*forge.CommitStatus, error) {
	return nil, forge.ErrNotSupported
}

func (s *unsupportedCheckRunService) List(context.Context, string, string, string) ([]forge.CheckRun, error) {
	return nil, forge.ErrNotSupported
}

func (s *unsupportedCommitService) ResolveCommit(context.Context, string, string, string) (string, error) {
	return "", forge.ErrNotSupported
}

func (s *unsupportedAttestationService) List(context.Context, string, string, string, forge.ListAttestationOpts) ([]forge.Attestation, error) {
	return nil, forge.ErrNotSupported
}

func (s *unsupportedAttestationService) Download(context.Context, string, string, forge.Attestation) (json.RawMessage, error) {
	return nil, forge.ErrNotSupported
}
