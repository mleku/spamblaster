package util

import (
	"encoding/hex"
	"fmt"
	"mleku.online/git/ec/schnorr"
	secp "mleku.online/git/ec/secp"
	"mleku.online/git/signr/pkg/nostr"
	"strings"
)

func NpubToHex(npub string) (pk string, err error) {
	var p *secp.PublicKey
	p, err = nostr.NpubToPublicKey(npub)
	if err != nil {
		err = fmt.Errorf("error decoding pubkey: %v", err)
	} else {
		pk = hex.EncodeToString(schnorr.SerializePubKey(p))
	}
	return
}

func DecodePub(npub string) (hexPub string, err error) {
	if strings.Contains(npub, "npub") {
		hexPub, err = NpubToHex(npub)
	}
	return
}
