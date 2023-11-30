package cmd

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	secp "mleku.online/git/ec/secp"
	"mleku.online/git/replicatr/pkg/nostr"
	"mleku.online/git/replicatr/pkg/nostr/kind"
	"mleku.online/git/signr/pkg/signr"
	"time"

	"github.com/spf13/cobra"
)

var wait int

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen <signing key name> <relay websocket address> [note ID to refer to as reply]",
	Short: "generate and random generated send event notes to a relay",
	Long: `random note generator for testing purposes.

fills in content with a random selection from the devlorem quote list, optionally  adds a reply reference.

web socket must have the proper wss:// or ws:// prefix as expected by the relay.
`,
	Run: func(cmd *cobra.Command, args []string) {
		var e error
		s, e = signr.Init(signr.PasswordEntryViaTTY)
		if e != nil {
			s.Fatal("fatal error: %s\n", e)
		}
		if len(args) < 1 {
			s.Fatal("<signing key name> argument is required\n")
		}
		if len(args) < 2 {
			s.Fatal("<relay websocket address> argument is required\n")
		}
		var sec *secp.SecretKey
		sec, e = s.GetKey(args[0], Pass)
		if e != nil {
			s.Fatal("error: %s\n", e)
		}

		// pick random quote to use in content field of event
		source := rand.Intn(len(quotes))
		quote := rand.Intn(len(quotes[source].Paragraphs))
		quoteText := fmt.Sprintf("\"%s\"\n- %s\n",
			quotes[source].Paragraphs[quote],
			quotes[source].Source)
		var tags nostr.Tags
		if len(args) > 2 {
			// if there is a note ID given, add its tag.
			tags = nostr.Tags{{"e", args[2], args[1], nostr.TagMarkerReply}}
		} else {
			// if no note ID is given this is a root
			tags = nostr.Tags{{"e", "", args[1], nostr.TagMarkerRoot}}
		}
		evt := nostr.Event{
			CreatedAt: nostr.Now(),
			Kind:      kind.TextNote,
			Tags:      tags,
			Content:   quoteText,
		}

		// tag note with an ID and sign it
		evt.ID = evt.GetID()
		e = evt.Sign(hex.EncodeToString(sec.Serialize()))
		if e != nil {
			s.Fatal("fatal error: %s\n", e)
		}
		b, _ := evt.MarshalJSON()
		// wait prescribed time before dispatching event.
		time.Sleep(time.Duration(wait) * time.Second)
		s.Info("signed event JSON:\n%s\n", string(b))
	},
}

func init() {
	rootCmd.AddCommand(genCmd)

	genCmd.PersistentFlags().IntVarP(&wait, "wait", "w", 1,
		"pause before dispatching event to relay")
}
