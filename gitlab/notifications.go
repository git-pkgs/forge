package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	forge "github.com/git-pkgs/forge"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabNotificationService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Notifications() forge.NotificationService {
	return &gitLabNotificationService{client: f.client}
}

func convertGitLabTodoTargetType(t string) forge.NotificationSubjectType {
	switch t {
	case "Issue":
		return forge.NotificationSubjectIssue
	case "MergeRequest":
		return forge.NotificationSubjectPullRequest
	case "Commit":
		return forge.NotificationSubjectCommit
	default:
		return forge.NotificationSubjectType(t)
	}
}

func convertGitLabTodo(t *gitlab.Todo) forge.Notification {
	result := forge.Notification{
		ID:     strconv.FormatInt(t.ID, 10),
		Title:  t.Body,
		Reason: string(t.ActionName),
		Unread: t.State == "pending",
	}

	result.SubjectType = convertGitLabTodoTargetType(string(t.TargetType))

	if t.Target != nil {
		result.URL = t.TargetURL
	}

	if t.Project != nil {
		result.Repo = t.Project.PathWithNamespace
		if result.Title == "" {
			result.Title = t.Project.Name
		}
	}

	if t.CreatedAt != nil {
		result.UpdatedAt = *t.CreatedAt
	}

	return result
}

func (s *gitLabNotificationService) List(ctx context.Context, opts forge.ListNotificationOpts) ([]forge.Notification, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	glOpts := &gitlab.ListTodosOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage), Page: int64(page)},
	}

	if opts.Unread {
		glOpts.State = gitlab.Ptr("pending")
	}

	if opts.Repo != "" {
		pid := opts.Repo
		projects, _, err := s.client.Projects.ListProjects(&gitlab.ListProjectsOptions{
			Search: gitlab.Ptr(pid),
		})
		if err == nil && len(projects) > 0 {
			for _, p := range projects {
				if p.PathWithNamespace == pid {
					glOpts.ProjectID = gitlab.Ptr(p.ID)
					break
				}
			}
		}
	}

	var all []forge.Notification
	for {
		todos, resp, err := s.client.Todos.ListTodos(glOpts)
		if err != nil {
			return nil, err
		}
		for _, t := range todos {
			all = append(all, convertGitLabTodo(t))
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		glOpts.Page = int64(resp.NextPage)
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *gitLabNotificationService) MarkRead(ctx context.Context, opts forge.MarkNotificationOpts) error {
	if opts.ID != "" {
		id, err := strconv.ParseInt(opts.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid notification ID: %s", opts.ID)
		}
		resp, err := s.client.Todos.MarkTodoAsDone(id)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return forge.ErrNotFound
			}
			return err
		}
		return nil
	}

	// GitLab's mark-all-done endpoint doesn't support filtering by project,
	// so for both repo-filtered and unfiltered we mark all.
	_, err := s.client.Todos.MarkAllTodosAsDone()
	return err
}

func (s *gitLabNotificationService) Get(ctx context.Context, id string) (*forge.Notification, error) {
	// GitLab has no single-todo GET endpoint. List with no filters and find it.
	todoID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid notification ID: %s", id)
	}

	todos, _, err := s.client.Todos.ListTodos(&gitlab.ListTodosOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, err
	}

	for _, t := range todos {
		if t.ID == todoID {
			result := convertGitLabTodo(t)
			return &result, nil
		}
	}

	return nil, forge.ErrNotFound
}
