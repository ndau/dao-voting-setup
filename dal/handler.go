package dal

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm/clause"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	// This blank import is required by the gorm implementation
	"github.com/ndau/dao-voting-setup/models"
	logger "github.com/ndau/go-logger"
)

const (
	dbType = "postgres"
	table  = "BPCDAO.accounts"
)

type Db struct {
	Client *gorm.DB
	Cfg    *models.Config
	Log    logger.Logger
}

// NewDb ...
func NewDb(cfg *models.Config, log logger.Logger) (*Db, error) {
	log.Info(cfg.ConnectionString)
	db, err := gorm.Open(postgres.Open(ParseURL(cfg.ConnectionString)))
	if err != nil {
		return nil, errors.Wrap(err, "Failed openning a DB connection")
	}
	// defer db.Close()

	return &Db{
		Client: db,
		Cfg:    cfg,
		Log:    log,
	}, nil
}

// Close ...
func (db *Db) Close() {
	db.Close()
}

// ListAccount - Read all existing accounts
func (db *Db) ListAccount() ([]models.VotingSetup, error) {
	accounts := []models.VotingSetup{}
	if err := db.Client.Order("address asc").Find(&accounts).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(err, "Failed reading from the accounts table")
		}
	}

	return accounts, nil
}

// InsertVotingList -
func (db *Db) UpsertVotingList(ctx context.Context, votings []models.VotingSetup) error {
	trackingNumber := ctx.Value("tracking_number").(string)
	db.Log.Infof("%s | Inserting '%d' account voting into the BPCDao.accounts table", trackingNumber, len(votings))

	return db.Client.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "address"}},
		DoUpdates: clause.AssignmentColumns([]string{"currency_seat_date", "votes"}),
	}).CreateInBatches(votings, 1000).Error
}

// ParseURL - Prase the DB connection string
func ParseURL(url string) string {
	s := strings.Split(url, ":")

	s1 := strings.Split(s[1], "/")
	username := s1[2]

	s2 := strings.Split(s[3], "/")
	port := s2[0]
	dbName := s2[1]

	s3 := strings.Split(s[2], "@")
	password := s3[0]
	dbHost := s3[1]

	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable password=%s", dbHost, port, username, dbName, password)
}
