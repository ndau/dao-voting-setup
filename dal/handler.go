package dal

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm/clause"

	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	// This blank import is required by the gorm implementation
	"github.com/ndau/dao-voting-setup/models"
	logger "github.com/ndau/go-logger"
	glogger "gorm.io/gorm/logger"
)

const (
	tblaccount  = "accounts"
	tblproposal = "proposals"
)

type Db struct {
	Client *gorm.DB
	Cfg    *models.Config
	Log    logger.Logger
}

// NewDb ...
func NewDb(cfg *models.Config, log logger.Logger) (*Db, error) {
	log.Info(cfg.ConnectionString)
	db, err := gorm.Open(postgres.Open(ParseURL(cfg.ConnectionString)), &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Info),
	})
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
		return nil, errors.Wrap(err, "failed reading from the accounts table")
	}

	return accounts, nil
}

// InsertVotingList -
func (db *Db) UpsertVotingList(ctx context.Context, votings []models.VotingSetup) error {
	trackingNumber := ctx.Value("tracking_number").(string)
	db.Log.Infof("%s | Inserting '%d' account voting into the accounts table", trackingNumber, len(votings))

	return db.Client.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "address"}},
		DoUpdates: clause.AssignmentColumns([]string{"currency_seat_date", "votes"}),
	}).CreateInBatches(votings, 1000).Error
}

// Unseat -
func (db *Db) Unseat(ctx context.Context, addresses []string) error {
	trackingNumber, _ := ctx.Value("tracking_number").(string)
	db.Log.Infof("%s | Try to update upto '%d' accounts that lost their seats, if existed", trackingNumber, len(addresses))

	if res := db.Client.Table(tblaccount).Where("address IN ?", addresses).Updates(map[string]interface{}{"currency_seat_date": "0001-01-01", "votes": 0.0}); res.Error == nil {
		db.Log.Infof("%s | Unseated '%d' accounts", trackingNumber, res.RowsAffected)
		return nil
	} else {
		return res.Error
	}
}

// ListActiveProposal - Read all existing accounts
func (db *Db) ListActiveProposal() ([]models.Proposal, error) {
	proposals := []models.Proposal{}
	if err := db.Client.Where("concluded IS NULL AND is_approved = TRUE").Order("closing_date asc").Find(&proposals).Error; err != nil {
		return nil, errors.Wrap(err, "failed reading from the accounts table")
	}

	return proposals, nil
}

// UpdateConcludedVotes -
func (db *Db) UpdateConcludedVotes(ctx context.Context, proposalId int64) error {
	trackingNumber, _ := ctx.Value("tracking_number").(string)
	db.Log.Infof("%s | Update concluded votes for proposal '%d'", trackingNumber, proposalId)

	if res := db.Client.Exec(`
		update public.votes v
		   set concluded_votes = COALESCE(a.votes,0)
			from public.accounts a
		 where v.user_address = a.address
		   and v.concluded_votes is null and v.proposal_id = ?`, proposalId); res.Error == nil {
		db.Log.Infof("%s | Updated '%d' concluded votes", trackingNumber, res.RowsAffected)
		return nil
	} else {
		return res.Error
	}
}

// ParseURL - Prase the DB connection string
func ParseURL(connectionString string) string {
	s := strings.Split(connectionString, ":")
	// schema := s[0]

	s1 := strings.Split(s[1], "/")
	username := s1[2]

	s2 := strings.Split(s[3], "/")
	port := s2[0]
	dbName := s2[1]

	s3 := strings.Split(s[2], "@")
	password := s3[0]
	dbHost := s3[1]

	// dns := url.URL{
	// 	User:     url.UserPassword(username, password),
	// 	Scheme:   schema,
	// 	Host:     fmt.Sprintf("%s:%s", dbHost, port),
	// 	Path:     dbName,
	// 	RawQuery: (&url.Values{"sslmode": []string{"disable"}}).Encode(),
	// }

	// fmt.Println(dns.String())

	// return dns.String()

	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable password=%s", dbHost, port, username, dbName, password)
}
