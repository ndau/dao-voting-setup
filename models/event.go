package models

// Event event
type Event struct {
	// TrackingNumber
	TrackingNumber string `json:"TrackingNumber,omitempty"`
}

// Data struct
type Data struct {
	// Network
	Network string `json:"Network"`

	// NodeAPI
	NodeAPI string `json:"NodeAPI"`

	// Limit
	Limit int `json:"Limit"`

	// StartAfterKey
	StartAfterKey string `json:"StartAfterKey"`
}
