package forges

import "strings"

// IssueState is the normalized lifecycle state for issues.
type IssueState string

const (
	IssueStateOpen    IssueState = "open"
	IssueStateClosed  IssueState = "closed"
	IssueStateUnknown IssueState = "unknown"
)

// PRStatus is the normalized lifecycle state for pull or merge requests.
type PRStatus string

const (
	PRStatusOpen    PRStatus = "open"
	PRStatusClosed  PRStatus = "closed"
	PRStatusMerged  PRStatus = "merged"
	PRStatusUnknown PRStatus = "unknown"
)

// CIStatus is the normalized execution state for CI runs and jobs.
type CIStatus string

const (
	CIStatusQueued    CIStatus = "queued"
	CIStatusRunning   CIStatus = "running"
	CIStatusCompleted CIStatus = "completed"
	CIStatusSuccess   CIStatus = "success"
	CIStatusFailed    CIStatus = "failed"
	CIStatusCancelled CIStatus = "cancelled"
	CIStatusSkipped   CIStatus = "skipped"
	CIStatusManual    CIStatus = "manual"
	CIStatusUnknown   CIStatus = "unknown"
)

// CIConclusion is the normalized result for completed CI runs and jobs.
type CIConclusion string

const (
	CIConclusionSuccess        CIConclusion = "success"
	CIConclusionFailure        CIConclusion = "failure"
	CIConclusionCancelled      CIConclusion = "cancelled"
	CIConclusionSkipped        CIConclusion = "skipped"
	CIConclusionNeutral        CIConclusion = "neutral"
	CIConclusionTimedOut       CIConclusion = "timed_out"
	CIConclusionActionRequired CIConclusion = "action_required"
	CIConclusionUnknown        CIConclusion = "unknown"
)

// CommitStatusState is the normalized state for commit status checks.
type CommitStatusState string

const (
	CommitStatusSuccess   CommitStatusState = "success"
	CommitStatusFailure   CommitStatusState = "failure"
	CommitStatusPending   CommitStatusState = "pending"
	CommitStatusError     CommitStatusState = "error"
	CommitStatusCancelled CommitStatusState = "cancelled"
	CommitStatusSkipped   CommitStatusState = "skipped"
	CommitStatusUnknown   CommitStatusState = "unknown"
)

// AccessLevel is the normalized permission level for repository collaborators.
type AccessLevel string

const (
	AccessLevelNone    AccessLevel = "none"
	AccessLevelRead    AccessLevel = "read"
	AccessLevelWrite   AccessLevel = "write"
	AccessLevelAdmin   AccessLevel = "admin"
	AccessLevelUnknown AccessLevel = "unknown"
)

// NormalizeIssueState maps forge-specific issue states to common states.
func NormalizeIssueState(state string) IssueState {
	switch normalizeToken(state) {
	case "open", "opened", "new", "reopened", "on_hold":
		return IssueStateOpen
	case "closed", "resolved", "declined", "rejected", "done", "invalid", "duplicate", "wontfix":
		return IssueStateClosed
	default:
		return IssueStateUnknown
	}
}

// NormalizePRStatus maps forge-specific pull request states to common states.
func NormalizePRStatus(state string) PRStatus {
	switch normalizeToken(state) {
	case "open", "opened", "new", "reopened":
		return PRStatusOpen
	case "closed", "declined", "rejected", "superseded":
		return PRStatusClosed
	case "merged":
		return PRStatusMerged
	default:
		return PRStatusUnknown
	}
}

// NormalizeCIStatus maps forge-specific CI run and job statuses to common states.
func NormalizeCIStatus(status string) CIStatus {
	switch normalizeToken(status) {
	case "queued", "pending", "created", "waiting", "requested", "scheduled":
		return CIStatusQueued
	case "running", "in_progress", "inprogress":
		return CIStatusRunning
	case "completed", "complete", "done", "finished":
		return CIStatusCompleted
	case "success", "successful", "succeeded", "passed", "passing", "approved":
		return CIStatusSuccess
	case "failed", "failure", "fail", "failing":
		return CIStatusFailed
	case "cancelled", "canceled", "canceling", "cancelling":
		return CIStatusCancelled
	case "skipped", "skip":
		return CIStatusSkipped
	case "manual", "blocked":
		return CIStatusManual
	default:
		return CIStatusUnknown
	}
}

// NormalizeCIConclusion maps forge-specific CI conclusions to common results.
func NormalizeCIConclusion(conclusion string) CIConclusion {
	switch normalizeToken(conclusion) {
	case "":
		return ""
	case "success", "successful", "succeeded", "passed", "passing", "approved":
		return CIConclusionSuccess
	case "failure", "failed", "fail", "failing", "declined", "rejected":
		return CIConclusionFailure
	case "cancelled", "canceled":
		return CIConclusionCancelled
	case "skipped", "skip":
		return CIConclusionSkipped
	case "neutral":
		return CIConclusionNeutral
	case "timed_out", "timedout", "timeout":
		return CIConclusionTimedOut
	case "action_required", "actionrequired":
		return CIConclusionActionRequired
	default:
		return CIConclusionUnknown
	}
}

// NormalizeCommitStatusState maps forge-specific commit status states to common states.
func NormalizeCommitStatusState(state string) CommitStatusState {
	switch normalizeToken(state) {
	case "success", "successful", "succeeded", "passed", "passing", "approved", "ok":
		return CommitStatusSuccess
	case "failure", "failed", "fail", "failing", "declined", "rejected":
		return CommitStatusFailure
	case "pending", "queued", "running", "in_progress", "created", "waiting":
		return CommitStatusPending
	case "error", "errored", "broken":
		return CommitStatusError
	case "cancelled", "canceled":
		return CommitStatusCancelled
	case "skipped", "skip":
		return CommitStatusSkipped
	default:
		return CommitStatusUnknown
	}
}

// NormalizeAccessLevel maps forge-specific collaborator permission names to common levels.
func NormalizeAccessLevel(permission string) AccessLevel {
	switch normalizeToken(permission) {
	case "", "none", "no_access":
		return AccessLevelNone
	case "read", "pull", "guest", "reporter", "triage", "planner":
		return AccessLevelRead
	case "write", "push", "developer":
		return AccessLevelWrite
	case "admin", "owner", "maintainer", "maintain":
		return AccessLevelAdmin
	default:
		return AccessLevelUnknown
	}
}

func normalizeToken(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, "-", "_")
	v = strings.ReplaceAll(v, " ", "_")
	return v
}
