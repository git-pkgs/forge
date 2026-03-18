package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

const maxErrorBodyLength = 200

var apiCmd = &cobra.Command{
	Use:   "api <endpoint>",
	Short: "Make an authenticated API request",
	Long:  "Makes an authenticated API request to the forge's API. The endpoint is relative to the API root.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		endpoint := args[0]

		_, owner, repoName, domain, err := resolve.Repo(flagRepo, flagForgeType)
		if err != nil {
			return err
		}

		// Template substitution
		endpoint = strings.ReplaceAll(endpoint, "{owner}", owner)
		endpoint = strings.ReplaceAll(endpoint, "{repo}", repoName)

		// Build full URL
		var baseURL string
		switch {
		case strings.Contains(domain, "github"):
			baseURL = "https://api." + domain
		case strings.Contains(domain, "gitlab"):
			baseURL = "https://" + domain + "/api/v4"
		case strings.Contains(domain, "bitbucket"):
			baseURL = "https://api.bitbucket.org/2.0"
		default:
			baseURL = "https://" + domain + "/api/v1"
		}

		url := baseURL + "/" + strings.TrimLeft(endpoint, "/")

		var body io.Reader
		if len(flagAPIFields) > 0 {
			data := make(map[string]string)
			for _, f := range flagAPIFields {
				k, v, ok := strings.Cut(f, "=")
				if !ok {
					return fmt.Errorf("invalid field format: %q (expected KEY=VALUE)", f)
				}
				data[k] = v
			}
			b, err := json.Marshal(data)
			if err != nil {
				return err
			}
			body = bytes.NewReader(b)
		}

		req, err := http.NewRequestWithContext(cmd.Context(), flagAPIMethod, url, body)
		if err != nil {
			return err
		}

		token := resolve.TokenForDomain(domain)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		for _, h := range flagAPIHeaders {
			k, v, ok := strings.Cut(h, ":")
			if !ok {
				return fmt.Errorf("invalid header format: %q (expected KEY:VALUE)", h)
			}
			req.Header.Set(strings.TrimSpace(k), strings.TrimSpace(v))
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()

		if flagAPIInclude {
			_, _ = fmt.Fprintf(os.Stdout, "%s %s\n", resp.Proto, resp.Status)
			for k, v := range resp.Header {
				_, _ = fmt.Fprintf(os.Stdout, "%s: %s\n", k, strings.Join(v, ", "))
			}
			_, _ = fmt.Fprintln(os.Stdout)
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if !flagAPISilent {
			// Try to pretty-print JSON
			var pretty bytes.Buffer
			if json.Indent(&pretty, respBody, "", "  ") == nil {
				_, _ = fmt.Fprintln(os.Stdout, pretty.String())
			} else {
				_, _ = os.Stdout.Write(respBody)
				_, _ = fmt.Fprintln(os.Stdout)
			}
		}

		if resp.StatusCode >= http.StatusBadRequest {
			msg := strings.TrimSpace(string(respBody))
			if len(msg) > maxErrorBodyLength {
				msg = msg[:maxErrorBodyLength]
			}
			if msg != "" {
				return fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
			}
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}

		return nil
	},
}

var (
	flagAPIMethod  string
	flagAPIFields  []string
	flagAPIHeaders []string
	flagAPIInclude bool
	flagAPISilent  bool
)

func init() {
	rootCmd.AddCommand(apiCmd)
	apiCmd.Flags().StringVarP(&flagAPIMethod, "method", "X", "GET", "HTTP method")
	apiCmd.Flags().StringArrayVarP(&flagAPIFields, "field", "f", nil, "Add a field (KEY=VALUE)")
	apiCmd.Flags().StringArrayVarP(&flagAPIHeaders, "header", "H", nil, "Add a header (KEY:VALUE)")
	apiCmd.Flags().BoolVarP(&flagAPIInclude, "include", "i", false, "Include HTTP response headers")
	apiCmd.Flags().BoolVar(&flagAPISilent, "silent", false, "Do not print response body")
}
