package creator

// Relay Creator Schema
type Relay struct {
	ID                   string      `json:"id"`
	Name                 string      `json:"name"`
	OwnerID              string      `json:"ownerId"`
	Status               interface{} `json:"status"`
	DefaultMessagePolicy bool        `json:"default_message_policy"`
	IP                   interface{} `json:"ip"`
	Capacity             interface{} `json:"capacity"`
	Port                 interface{} `json:"port"`
	AllowList            struct {
		ID           string `json:"id"`
		RelayID      string `json:"relayId"`
		ListKeywords []struct {
			ID          string      `json:"id"`
			AllowListID string      `json:"AllowListId"`
			BlockListID interface{} `json:"BlockListId"`
			Keyword     string      `json:"keyword"`
			Reason      string      `json:"reason"`
			ExpiresAt   interface{} `json:"expires_at"`
		} `json:"list_keywords"`
		ListPubkeys []struct {
			ID          string      `json:"id"`
			AllowListID string      `json:"AllowListId"`
			BlockListID interface{} `json:"BlockListId"`
			Pubkey      string      `json:"pubkey"`
			Reason      string      `json:"reason"`
			ExpiresAt   interface{} `json:"expires_at"`
		} `json:"list_pubkeys"`
	} `json:"allow_list"`
	BlockList struct {
		ID           string `json:"id"`
		RelayID      string `json:"relayId"`
		ListKeywords []struct {
			ID          string      `json:"id"`
			AllowListID interface{} `json:"AllowListId"`
			BlockListID string      `json:"BlockListId"`
			Keyword     string      `json:"keyword"`
			Reason      string      `json:"reason"`
			ExpiresAt   interface{} `json:"expires_at"`
		} `json:"list_keywords"`
		ListPubkeys []struct {
			ID          string      `json:"id"`
			AllowListID interface{} `json:"AllowListId"`
			BlockListID string      `json:"BlockListId"`
			Pubkey      string      `json:"pubkey"`
			Reason      string      `json:"reason"`
			ExpiresAt   interface{} `json:"expires_at"`
		} `json:"list_pubkeys"`
	} `json:"block_list"`
	Owner struct {
		ID     string      `json:"id"`
		Pubkey string      `json:"pubkey"`
		Name   interface{} `json:"name"`
	} `json:"owner"`

	Moderators []struct {
		ID      string `json:"id"`
		RelayID string `json:"relayId"`
		UserID  string `json:"userId"`
		User    struct {
			Pubkey string `json:"pubkey"`
		} `json:"user"`
	} `json:"moderators"`
}
