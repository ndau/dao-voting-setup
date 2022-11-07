package models

import (
	"time"
)

// VotingSetup -
type VotingSetup struct {
	Address          string
	CurrencySeatDate time.Time
	Votes            float64
}

// TableName - Return table name
func (t VotingSetup) TableName() string {
	return "accounts"
}
