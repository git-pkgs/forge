package forges

import "testing"

func TestNormalizePRStatus(t *testing.T) {
	tests := []struct {
		in   string
		want PRStatus
	}{
		{"open", PRStatusOpen},
		{"opened", PRStatusOpen},
		{"MERGED", PRStatusMerged},
		{"DECLINED", PRStatusClosed},
		{"superseded", PRStatusClosed},
		{"unexpected", PRStatusUnknown},
	}

	for _, tt := range tests {
		if got := NormalizePRStatus(tt.in); got != tt.want {
			t.Errorf("NormalizePRStatus(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeCommitStatusState(t *testing.T) {
	tests := []struct {
		in   string
		want CommitStatusState
	}{
		{"success", CommitStatusSuccess},
		{"passing", CommitStatusSuccess},
		{"approved", CommitStatusSuccess},
		{"failed", CommitStatusFailure},
		{"in_progress", CommitStatusPending},
		{"errored", CommitStatusError},
		{"canceled", CommitStatusCancelled},
		{"unexpected", CommitStatusUnknown},
	}

	for _, tt := range tests {
		if got := NormalizeCommitStatusState(tt.in); got != tt.want {
			t.Errorf("NormalizeCommitStatusState(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeAccessLevel(t *testing.T) {
	tests := []struct {
		in   string
		want AccessLevel
	}{
		{"pull", AccessLevelRead},
		{"reporter", AccessLevelRead},
		{"push", AccessLevelWrite},
		{"developer", AccessLevelWrite},
		{"maintainer", AccessLevelAdmin},
		{"owner", AccessLevelAdmin},
		{"", AccessLevelNone},
		{"custom", AccessLevelUnknown},
	}

	for _, tt := range tests {
		if got := NormalizeAccessLevel(tt.in); got != tt.want {
			t.Errorf("NormalizeAccessLevel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeCIStatusAndConclusion(t *testing.T) {
	if got := NormalizeCIStatus("in_progress"); got != CIStatusRunning {
		t.Errorf("NormalizeCIStatus(in_progress) = %q", got)
	}
	if got := NormalizeCIStatus("canceled"); got != CIStatusCancelled {
		t.Errorf("NormalizeCIStatus(canceled) = %q", got)
	}
	if got := NormalizeCIConclusion("timed-out"); got != CIConclusionTimedOut {
		t.Errorf("NormalizeCIConclusion(timed-out) = %q", got)
	}
	if got := NormalizeCIConclusion(""); got != "" {
		t.Errorf("NormalizeCIConclusion(empty) = %q", got)
	}
}
