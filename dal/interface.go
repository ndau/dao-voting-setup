package dal

import (
	"context"

	"github.com/ndau/dao-voting-setup/models"
)

//go:generate mockgen -destination=./mocks/mock_repo.go -package=mocks github.com/ndau/dao-voting-setup/dal Repo
type Repo interface {
	Close()
	ListAccount() ([]models.VotingSetup, error)
	Unseat(ctx context.Context, addresses []string) error
	UpsertVotingList(ctx context.Context, votings []models.VotingSetup) error
}
