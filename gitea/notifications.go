package gitea

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
	forge "github.com/git-pkgs/forge"
)

const ownerRepoParts = 2

type giteaNotificationService struct {
	client *gitea.Client
}

func (f *giteaForge) Notifications() forge.NotificationService {
	return &giteaNotificationService{client: f.client}
}

func convertGiteaSubjectType(t string) forge.NotificationSubjectType {
	switch t {
	case "Issue":
		return forge.NotificationSubjectIssue
	case "Pull":
		return forge.NotificationSubjectPullRequest
	case "Commit":
		return forge.NotificationSubjectCommit
	case "Repository":
		return forge.NotificationSubjectRepository
	default:
		return forge.NotificationSubjectType(strings.ToLower(t))
	}
}

func convertGiteaNotification(n *gitea.NotificationThread) forge.Notification {
	result := forge.Notification{
		ID:     strconv.FormatInt(n.ID, 10),
		Unread: n.Unread,
	}

	if n.Subject != nil {
		result.Title = n.Subject.Title
		result.SubjectType = convertGiteaSubjectType(string(n.Subject.Type))
		result.URL = n.Subject.HTMLURL
	}

	if n.Repository != nil {
		result.Repo = n.Repository.FullName
	}

	if !n.UpdatedAt.IsZero() {
		result.UpdatedAt = n.UpdatedAt
	}

	return result
}

func (s *giteaNotificationService) List(ctx context.Context, opts forge.ListNotificationOpts) ([]forge.Notification, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	statuses := []gitea.NotifyStatus{}
	if opts.Unread {
		statuses = append(statuses, gitea.NotifyStatusUnread)
	}

	var all []forge.Notification
	var err error

	if opts.Repo != "" {
		parts := strings.SplitN(opts.Repo, "/", ownerRepoParts)
		if len(parts) == ownerRepoParts {
			all, err = s.listRepoNotifications(parts[0], parts[1], page, perPage, statuses, opts.Limit)
		}
	} else {
		all, err = s.listAllNotifications(page, perPage, statuses, opts.Limit)
	}
	if err != nil {
		return nil, err
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *giteaNotificationService) listRepoNotifications(owner, repo string, page, perPage int, statuses []gitea.NotifyStatus, limit int) ([]forge.Notification, error) {
	var all []forge.Notification
	for {
		notifications, resp, err := s.client.ListRepoNotifications(owner, repo, gitea.ListNotificationOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
			Status:      statuses,
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, n := range notifications {
			all = append(all, convertGiteaNotification(n))
		}
		if len(notifications) < perPage || (limit > 0 && len(all) >= limit) {
			break
		}
		page++
	}
	return all, nil
}

func (s *giteaNotificationService) listAllNotifications(page, perPage int, statuses []gitea.NotifyStatus, limit int) ([]forge.Notification, error) {
	var all []forge.Notification
	for {
		notifications, _, err := s.client.ListNotifications(gitea.ListNotificationOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
			Status:      statuses,
		})
		if err != nil {
			return nil, err
		}
		for _, n := range notifications {
			all = append(all, convertGiteaNotification(n))
		}
		if len(notifications) < perPage || (limit > 0 && len(all) >= limit) {
			break
		}
		page++
	}
	return all, nil
}

func (s *giteaNotificationService) MarkRead(ctx context.Context, opts forge.MarkNotificationOpts) error {
	if opts.ID != "" {
		id, err := strconv.ParseInt(opts.ID, 10, 64)
		if err != nil {
			return err
		}
		_, resp, err := s.client.ReadNotification(id)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return forge.ErrNotFound
			}
			return err
		}
		return nil
	}

	if opts.Repo != "" {
		parts := strings.SplitN(opts.Repo, "/", ownerRepoParts)
		if len(parts) == ownerRepoParts {
			_, resp, err := s.client.ReadRepoNotifications(parts[0], parts[1], gitea.MarkNotificationOptions{})
			if err != nil {
				if resp != nil && resp.StatusCode == http.StatusNotFound {
					return forge.ErrNotFound
				}
				return err
			}
			return nil
		}
	}

	_, _, err := s.client.ReadNotifications(gitea.MarkNotificationOptions{})
	return err
}

func (s *giteaNotificationService) Get(ctx context.Context, id string) (*forge.Notification, error) {
	nID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, err
	}

	n, resp, err := s.client.GetNotification(nID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	result := convertGiteaNotification(n)
	return &result, nil
}
