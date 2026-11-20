package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	forge "github.com/git-pkgs/forge"
)

const deployKeyTestKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKoJlWbTSIfrkeBd3Ti0FbfIEe4kmYVnrIBzMrVAzuhk"

func deployKeyJSON(id int64, label string) map[string]any {
	return map[string]any{
		"type":       "deploy_key",
		"id":         id,
		"key":        deployKeyTestKey,
		"label":      label,
		"comment":    "ci@example.com",
		"created_on": "2024-03-05T10:20:30.123456+00:00",
		"last_used":  nil,
	}
}

func TestBitbucketListDeployKeys(t *testing.T) {
	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/deploy-keys", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want bearer token", got)
		}
		if r.URL.Query().Get("page") == "" {
			assertEqual(t, "pagelen", "100", r.URL.Query().Get("pagelen"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []any{deployKeyJSON(11, "ci")},
				"next":   srvURL + "/2.0/repositories/atlassian/stash/deploy-keys?pagelen=100&page=2",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []any{deployKeyJSON(12, "mirror")},
		})
	})
	f, u := newTestForge(t, mux)
	srvURL = u

	keys, err := f.DeployKeys().List(context.Background(), "atlassian", "stash", forge.ListDeployKeyOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	created := time.Date(2024, 3, 5, 10, 20, 30, 123456000, time.UTC)
	want := []forge.DeployKey{
		{ID: 11, Title: "ci", Key: deployKeyTestKey, ReadOnly: true, CreatedAt: created},
		{ID: 12, Title: "mirror", Key: deployKeyTestKey, ReadOnly: true, CreatedAt: created},
	}
	if len(keys) != len(want) {
		t.Fatalf("got %d keys, want %d: %+v", len(keys), len(want), keys)
	}
	for i := range want {
		if keys[i].ID != want[i].ID || keys[i].Title != want[i].Title || keys[i].Key != want[i].Key ||
			keys[i].ReadOnly != want[i].ReadOnly || !keys[i].CreatedAt.Equal(want[i].CreatedAt) {
			t.Errorf("keys[%d] = %+v, want %+v", i, keys[i], want[i])
		}
	}
}

func TestBitbucketListDeployKeysLimit(t *testing.T) {
	var requests int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/deploy-keys", func(w http.ResponseWriter, r *http.Request) {
		requests++
		assertEqual(t, "pagelen", "1", r.URL.Query().Get("pagelen"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []any{deployKeyJSON(11, "ci")},
			"next":   "http://unused.invalid/next",
		})
	})
	f, _ := newTestForge(t, mux)

	keys, err := f.DeployKeys().List(context.Background(), "atlassian", "stash", forge.ListDeployKeyOpts{Limit: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualInt(t, "len", 1, len(keys))
	assertEqualInt(t, "requests", 1, requests)
}

func TestBitbucketGetDeployKey(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/deploy-keys/11", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(deployKeyJSON(11, "ci"))
	})
	f, _ := newTestForge(t, mux)

	key, err := f.DeployKeys().Get(context.Background(), "atlassian", "stash", 11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualInt(t, "ID", 11, int(key.ID))
	assertEqual(t, "Title", "ci", key.Title)
	assertEqual(t, "Key", deployKeyTestKey, key.Key)
	assertEqualBool(t, "ReadOnly", true, key.ReadOnly)

	if _, err := f.DeployKeys().Get(context.Background(), "atlassian", "stash", 99); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("Get missing key: err = %v, want ErrNotFound", err)
	}
}

func TestBitbucketCreateDeployKey(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2.0/repositories/atlassian/stash/deploy-keys", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding body: %v", err)
		}
		want := map[string]any{"key": deployKeyTestKey, "label": "ci"}
		if len(body) != len(want) || body["key"] != want["key"] || body["label"] != want["label"] {
			t.Errorf("body = %v, want %v", body, want)
		}
		_ = json.NewEncoder(w).Encode(deployKeyJSON(11, "ci"))
	})
	f, _ := newTestForge(t, mux)

	key, err := f.DeployKeys().Create(context.Background(), "atlassian", "stash", forge.CreateDeployKeyOpts{
		Title:    "ci",
		Key:      deployKeyTestKey,
		ReadOnly: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualInt(t, "ID", 11, int(key.ID))
	assertEqual(t, "Title", "ci", key.Title)
	assertEqualBool(t, "ReadOnly", true, key.ReadOnly)
}

func TestBitbucketCreateDeployKeyWithWriteAccess(t *testing.T) {
	f, _ := newTestForge(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	}))

	_, err := f.DeployKeys().Create(context.Background(), "atlassian", "stash", forge.CreateDeployKeyOpts{
		Title: "ci",
		Key:   deployKeyTestKey,
	})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("err = %v, want ErrNotSupported", err)
	}
}

func TestBitbucketDeleteDeployKey(t *testing.T) {
	var deleted bool
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /2.0/repositories/atlassian/stash/deploy-keys/11", func(w http.ResponseWriter, r *http.Request) {
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})
	f, _ := newTestForge(t, mux)

	if err := f.DeployKeys().Delete(context.Background(), "atlassian", "stash", 11); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualBool(t, "deleted", true, deleted)

	if err := f.DeployKeys().Delete(context.Background(), "atlassian", "stash", 99); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("Delete missing key: err = %v, want ErrNotFound", err)
	}
}
