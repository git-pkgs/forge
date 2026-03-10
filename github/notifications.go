package github

import (
	"context"
	"net/http"
	"strings"

	forge "github.com/git-pkgs/forge"
	"github.com/google/go-github/v82/github"
)

type gitHubNotificationService struct {
	client *github.Client
}

func (f *gitHubForge) Notifications() forge.NotificationService {
	return &gitHubNotificationService{client: f.client}
}

func convertGitHubSubjectType(t string) forge.NotificationSubjectType {
	switch t {
	case "Issue":
		return forge.NotificationSubjectIssue
	case "PullRequest":
		return forge.NotificationSubjectPullRequest
	case "Commit":
		return forge.NotificationSubjectCommit
	case "Release":
		return forge.NotificationSubjectRelease
	case "Discussion":
		return forge.NotificationSubjectDiscussion
	case "RepositoryVulnerabilityAlert", "RepositoryDependabotAlertsThread":
		return forge.NotificationSubjectRepository
	default:
		return forge.NotificationSubjectType(strings.ToLower(t))
	}
}

func convertGitHubNotification(n *github.Notification) forge.Notification {
	result := forge.Notification{
		ID:     n.GetID(),
		Unread: n.GetUnread(),
		Reason: n.GetReason(),
	}

	if r := n.GetRepository(); r != nil {
		result.Repo = r.GetFullName()
		result.URL = r.GetHTMLURL()
	}

	if s := n.GetSubject(); s != nil {
		result.Title = s.GetTitle()
		result.SubjectType = convertGitHubSubjectType(s.GetType())
	}

	if t := n.GetUpdatedAt(); !t.IsZero() {
		result.UpdatedAt = t.Time
	}

	return result
}

func (s *gitHubNotificationService) List(ctx context.Context, opts forge.ListNotificationOpts) ([]forge.Notification, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.NotificationListOptions{
		All:         !opts.Unread,
		ListOptions: github.ListOptions{PerPage: perPage, Page: page},
	}

	var all []forge.Notification

	if opts.Repo != "" {
		parts := strings.SplitN(opts.Repo, "/", 2)
		if len(parts) == 2 {
			for {
				notifications, resp, err := s.client.Activity.ListRepositoryNotifications(ctx, parts[0], parts[1], ghOpts)
				if err != nil {
					if resp != nil && resp.StatusCode == http.StatusNotFound {
						return nil, forge.ErrNotFound
					}
					return nil, err
				}
				for _, n := range notifications {
					all = append(all, convertGitHubNotification(n))
				}
				if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
					break
				}
				ghOpts.Page = resp.NextPage
			}
		}
	} else {
		for {
			notifications, resp, err := s.client.Activity.ListNotifications(ctx, ghOpts)
			if err != nil {
				return nil, err
			}
			for _, n := range notifications {
				all = append(all, convertGitHubNotification(n))
			}
			if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
				break
			}
			ghOpts.Page = resp.NextPage
		}
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *gitHubNotificationService) MarkRead(ctx context.Context, opts forge.MarkNotificationOpts) error {
	if opts.ID != "" {
		resp, err := s.client.Activity.MarkThreadRead(ctx, opts.ID)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return forge.ErrNotFound
			}
			return err
		}
		return nil
	}

	if opts.Repo != "" {
		parts := strings.SplitN(opts.Repo, "/", 2)
		if len(parts) == 2 {
			resp, err := s.client.Activity.MarkRepositoryNotificationsRead(ctx, parts[0], parts[1], github.Timestamp{})
			if err != nil {
				if resp != nil && resp.StatusCode == http.StatusNotFound {
					return forge.ErrNotFound
				}
				return err
			}
			return nil
		}
	}

	resp, err := s.client.Activity.MarkNotificationsRead(ctx, github.Timestamp{})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubNotificationService) Get(ctx context.Context, id string) (*forge.Notification, error) {
	n, resp, err := s.client.Activity.GetThread(ctx, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubNotification(n)
	return &result, nil
}
