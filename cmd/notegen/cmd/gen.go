package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"math/rand"
	secp "mleku.online/git/ec/secp"
	"mleku.online/git/replicatr/pkg/nostr/kind"
	"mleku.online/git/replicatr/pkg/nostr/nip1"
	"mleku.online/git/replicatr/pkg/nostr/tag"
	"mleku.online/git/replicatr/pkg/nostr/tags"
	ts "mleku.online/git/replicatr/pkg/nostr/time"
	"mleku.online/git/signr/pkg/signr"
	"time"
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
		src := rand.Intn(len(quotes))
		q := rand.Intn(len(quotes[src].Paragraphs))
		quoteText := fmt.Sprintf("\"%s\"\n- %s\n",
			quotes[src].Paragraphs[q],
			quotes[src].Source)
		var t tags.T
		if len(args) > 2 {
			// if there is a note ID given, add its tag.
			t = tags.T{{"e", args[2], args[1], tag.MarkerReply}}
		} else {
			// if no note ID is given this is a root
			t = tags.T{{"e", "", args[1], tag.MarkerRoot}}
		}
		ev := &nip1.Event{
			CreatedAt: ts.Now(),
			Kind:      kind.TextNote,
			Tags:      t,
			Content:   quoteText,
		}

		// tag note with an ID and sign it
		ev.ID = ev.GetID()
		e = ev.Sign(hex.EncodeToString(sec.Serialize()))
		if e != nil {
			s.Fatal("fatal error: %s\n", e)
		}
		b, _ := json.Marshal(ev)
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
