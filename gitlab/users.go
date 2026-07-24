package gitlab

import (
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func resolveUserIDs(client *gitlab.Client, usernames []string) ([]int64, error) {
	ids := make([]int64, 0, len(usernames))
	for _, username := range usernames {
		users, _, err := client.Users.ListUsers(&gitlab.ListUsersOptions{
			Username: gitlab.Ptr(username),
		})
		if err != nil {
			return nil, fmt.Errorf("looking up user %q: %w", username, err)
		}
		if len(users) == 0 {
			return nil, fmt.Errorf("user %q not found", username)
		}
		ids = append(ids, int64(users[0].ID))
	}
	return ids, nil
}
