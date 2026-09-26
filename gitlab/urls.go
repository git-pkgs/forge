package gitlab

func (s *gitLabRepoService) SettingsURL(repoHTMLURL string) string {
	return repoHTMLURL + "/-/settings"
}

func (s *gitLabRepoService) WikiURL(repoHTMLURL string) string {
	return repoHTMLURL + "/-/wikis"
}

func (s *gitLabRepoService) ActionsURL(repoHTMLURL string) string {
	return repoHTMLURL + "/-/pipelines"
}

func (s *gitLabRepoService) ReleasesURL(repoHTMLURL string) string {
	return repoHTMLURL + "/-/releases"
}

func (s *gitLabRepoService) BlobURL(repoHTMLURL, ref, path string) string {
	return repoHTMLURL + "/-/blob/" + ref + "/" + path
}

func (s *gitLabIssueService) ListURL(repoHTMLURL string) string {
	return repoHTMLURL + "/-/issues"
}

func (s *gitLabLabelService) ListURL(repoHTMLURL string) string {
	return repoHTMLURL + "/-/labels"
}

func (s *gitLabPRService) ListURL(repoHTMLURL string) string {
	return repoHTMLURL + "/-/merge_requests"
}
