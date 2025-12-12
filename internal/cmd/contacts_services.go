package cmd

import (
	"context"

	"github.com/steipete/gogcli/internal/googleapi"
	"google.golang.org/api/people/v1"
)

var (
	newPeopleContactsService      func(ctx context.Context, email string) (*people.Service, error) = googleapi.NewPeopleContacts
	newPeopleOtherContactsService func(ctx context.Context, email string) (*people.Service, error) = googleapi.NewPeopleOtherContacts
	newPeopleDirectoryService     func(ctx context.Context, email string) (*people.Service, error) = googleapi.NewPeopleDirectory
)
