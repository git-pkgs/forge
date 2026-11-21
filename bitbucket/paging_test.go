package bitbucket

import "testing"

func TestPageQuery(t *testing.T) {
	tests := []struct {
		name                 string
		page, perPage, limit int
		want                 string
	}{
		{"defaults", 0, 0, 0, "pagelen=100"},
		{"per page", 0, 25, 0, "pagelen=25"},
		{"per page capped", 0, 500, 0, "pagelen=100"},
		{"limit shrinks first page", 0, 0, 10, "pagelen=10"},
		{"limit above per page", 1, 25, 60, "pagelen=25"},
		{"start page", 3, 20, 0, "pagelen=20&page=3"},
		{"start page keeps per page under limit", 2, 50, 10, "pagelen=50&page=2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, "query", tt.want, pageQuery(tt.page, tt.perPage, tt.limit))
		})
	}
}
