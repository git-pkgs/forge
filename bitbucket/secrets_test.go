package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"testing"

	forge "github.com/git-pkgs/forge"
)

const (
	variablesPath   = "/2.0/repositories/atlassian/stash/pipelines_config/variables"
	tokenVarUUID    = "{8c2a4f1e-0b7d-4e44-9a43-1f5d3c2b6a10}"
	tokenVarEscaped = "%7B8c2a4f1e-0b7d-4e44-9a43-1f5d3c2b6a10%7D"
)

func variableJSON(uuid, key string, secured bool) map[string]any {
	v := map[string]any{
		"type":    "pipeline_variable",
		"uuid":    uuid,
		"key":     key,
		"secured": secured,
	}
	if !secured {
		v["value"] = "visible"
	}
	return v
}

// variablesHandler serves the variable list in two pages, with the
// DEPLOY_TOKEN variable on the second.
func variablesHandler(t *testing.T, srvURL *string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want bearer token", got)
		}
		if r.URL.Query().Get("page") == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []any{variableJSON("{11111111-1111-1111-1111-111111111111}", "REGION", false)},
				"next":   *srvURL + variablesPath + "?pagelen=100&page=2",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []any{variableJSON(tokenVarUUID, "DEPLOY_TOKEN", true)},
		})
	}
}

func TestBitbucketListSecrets(t *testing.T) {
	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+variablesPath, variablesHandler(t, &srvURL))
	f, u := newTestForge(t, mux)
	srvURL = u

	secrets, err := f.Secrets().List(context.Background(), "atlassian", "stash", forge.ListSecretOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []forge.Secret{{Name: "REGION"}, {Name: "DEPLOY_TOKEN"}}
	if !slices.Equal(secrets, want) {
		t.Errorf("secrets = %+v, want %+v", secrets, want)
	}
}

func TestBitbucketListSecretsLimit(t *testing.T) {
	var srvURL string
	var requests int
	mux := http.NewServeMux()
	list := variablesHandler(t, &srvURL)
	mux.HandleFunc("GET "+variablesPath, func(w http.ResponseWriter, r *http.Request) {
		requests++
		assertEqual(t, "pagelen", "1", r.URL.Query().Get("pagelen"))
		list(w, r)
	})
	f, u := newTestForge(t, mux)
	srvURL = u

	secrets, err := f.Secrets().List(context.Background(), "atlassian", "stash", forge.ListSecretOpts{Limit: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualInt(t, "len", 1, len(secrets))
	assertEqualInt(t, "requests", 1, requests)
}

func decodeVariableBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	return body
}

func assertSecuredVariableBody(t *testing.T, body map[string]any, key, value string) {
	t.Helper()
	if len(body) != 3 || body["key"] != key || body["value"] != value || body["secured"] != true {
		t.Errorf("body = %v, want key %q, value %q, secured", body, key, value)
	}
}

func TestBitbucketSetSecretCreates(t *testing.T) {
	var srvURL string
	var created bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+variablesPath, variablesHandler(t, &srvURL))
	mux.HandleFunc("POST "+variablesPath, func(w http.ResponseWriter, r *http.Request) {
		created = true
		assertSecuredVariableBody(t, decodeVariableBody(t, r), "NPM_TOKEN", "s3cret")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(variableJSON("{22222222-2222-2222-2222-222222222222}", "NPM_TOKEN", true))
	})
	mux.HandleFunc("PUT "+variablesPath+"/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected update of %s", r.PathValue("uuid"))
	})
	f, u := newTestForge(t, mux)
	srvURL = u

	err := f.Secrets().Set(context.Background(), "atlassian", "stash", forge.SetSecretOpts{Name: "NPM_TOKEN", Value: "s3cret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualBool(t, "created", true, created)
}

func TestBitbucketSetSecretUpdates(t *testing.T) {
	var srvURL string
	var updated bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+variablesPath, variablesHandler(t, &srvURL))
	mux.HandleFunc("POST "+variablesPath, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected create")
	})
	mux.HandleFunc("PUT "+variablesPath+"/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		updated = true
		assertEqual(t, "uuid", tokenVarUUID, r.PathValue("uuid"))
		assertEqual(t, "escaped path", variablesPath+"/"+tokenVarEscaped, r.URL.EscapedPath())
		assertSecuredVariableBody(t, decodeVariableBody(t, r), "DEPLOY_TOKEN", "rotated")
		_ = json.NewEncoder(w).Encode(variableJSON(tokenVarUUID, "DEPLOY_TOKEN", true))
	})
	f, u := newTestForge(t, mux)
	srvURL = u

	err := f.Secrets().Set(context.Background(), "atlassian", "stash", forge.SetSecretOpts{Name: "DEPLOY_TOKEN", Value: "rotated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualBool(t, "updated", true, updated)
}

func TestBitbucketSetSecretListError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+variablesPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	mux.HandleFunc("POST "+variablesPath, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected create after a failed lookup")
	})
	f, _ := newTestForge(t, mux)

	err := f.Secrets().Set(context.Background(), "atlassian", "stash", forge.SetSecretOpts{Name: "NPM_TOKEN", Value: "s3cret"})
	if !errors.Is(err, forge.ErrUnauthorized) {
		t.Errorf("err = %v, want ErrUnauthorized", err)
	}
}

func TestBitbucketDeleteSecret(t *testing.T) {
	var srvURL string
	var deleted bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+variablesPath, variablesHandler(t, &srvURL))
	mux.HandleFunc("DELETE "+variablesPath+"/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		deleted = true
		assertEqual(t, "uuid", tokenVarUUID, r.PathValue("uuid"))
		assertEqual(t, "escaped path", variablesPath+"/"+tokenVarEscaped, r.URL.EscapedPath())
		w.WriteHeader(http.StatusNoContent)
	})
	f, u := newTestForge(t, mux)
	srvURL = u

	if err := f.Secrets().Delete(context.Background(), "atlassian", "stash", "DEPLOY_TOKEN"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualBool(t, "deleted", true, deleted)
}

func TestBitbucketDeleteSecretNotFound(t *testing.T) {
	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+variablesPath, variablesHandler(t, &srvURL))
	mux.HandleFunc("DELETE "+variablesPath+"/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected delete of %s", r.PathValue("uuid"))
	})
	f, u := newTestForge(t, mux)
	srvURL = u

	err := f.Secrets().Delete(context.Background(), "atlassian", "stash", "MISSING")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
