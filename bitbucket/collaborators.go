package bitbucket

import (
	"context"

	forge "github.com/git-pkgs/forge"
)

type bitbucketCollaboratorService struct{}

func (f *bitbucketForge) Collaborators() forge.CollaboratorService {
	return &bitbucketCollaboratorService{}
}

func (s *bitbucketCollaboratorService) List(_ context.Context, _, _ string, _ forge.ListCollaboratorOpts) ([]forge.Collaborator, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketCollaboratorService) Add(_ context.Context, _, _, _ string, _ forge.AddCollaboratorOpts) error {
	return forge.ErrNotSupported
}

func (s *bitbucketCollaboratorService) Remove(_ context.Context, _, _, _ string) error {
	return forge.ErrNotSupported
}
