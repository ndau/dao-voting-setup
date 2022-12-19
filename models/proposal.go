package models

import (
	"time"
)

// Proposal -
type Proposal struct {
	ProposalID  int64
	IsApproved  bool
	ClosingDate time.Time
	Concluded   bool
}

// TableName - Return table name
func (t Proposal) TableName() string {
	return "proposals"
}
