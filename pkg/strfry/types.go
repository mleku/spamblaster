package strfry

import "github.com/mleku/replicatr/pkg/nostr"

// Event is the JSON format of events (from stdin)
type Event struct {
	Event      nostr.Event `json:"event"`
	ReceivedAt int         `json:"receivedAt"`
	SourceInfo string      `json:"sourceInfo"`
	SourceType string      `json:"sourceType"`
	Type       string      `json:"type"`
}

// Result are instructions for Strfry from a plugin in response to an Event
type Result struct {
	ID     string `json:"id"`     // event id
	Action string `json:"action"` // accept or reject
	Msg    string `json:"msg"`    // sent to client for reject
}
