package cmd

import (
	"encoding/hex"
	"fmt"
	"github.com/spf13/viper"
	"math/rand"
	secp256k1 "mleku.online/git/ec/secp"
	"mleku.online/git/replicatr/pkg/nostr"
	"mleku.online/git/replicatr/pkg/nostr/kind"
	"mleku.online/git/signr/pkg/signr"
	"os"

	"github.com/spf13/cobra"
)

var (
	Pass string
	s    *signr.Signr
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "notegen <signing key name>",
	Short: "generate and random generated send event notes to a relay",
	Long: `random note generator for testing purposes.

fills in content with a random selection from the devlorem quote list, optionally  adds a reply reference.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		var e error
		s, e = signr.Init(signr.PasswordEntryViaTTY)
		if e != nil {
			s.Fatal("fatal error: %s\n", e)
		}
		if len(args) < 1 {
			s.Fatal("<signing key name> argument is required\n")
		}
		var sec *secp256k1.SecretKey
		sec, e = s.GetKey(args[0], Pass)
		if e != nil {
			s.Fatal("error: %s\n", e)
		}

		source := rand.Intn(len(quotes))
		quote := rand.Intn(len(quotes[source].Paragraphs))
		quoteText := fmt.Sprintf("\"%s\"\n- %s\n",
			quotes[source].Paragraphs[quote],
			quotes[source].Source)
		evt := nostr.Event{
			Kind:    kind.TextNote,
			Content: quoteText,
		}
		evt.ID = evt.GetID()
		e = evt.Sign(hex.EncodeToString(sec.Serialize()))
		if e != nil {
			s.Fatal("fatal error: %s\n", e)
		}
		b, _ := evt.MarshalJSON()
		fmt.Println(string(b))
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	var err error
	// because this is a CLI app we know the user can enter passwords this way.
	// other types of apps using this can load the environment variables.
	s, err = signr.Init(signr.PasswordEntryViaTTY)
	if err != nil {
		s.Fatal("fatal error: %s\n", err)
	}
	rootCmd.PersistentFlags().BoolVarP(&s.Verbose,
		"verbose", "v", false, "prints more things")
	rootCmd.PersistentFlags().BoolVarP(&s.Color,
		"color", "c", false, "prints color things")
	cobra.OnInitialize(initConfig(s))
}

// initConfig reads in config file and ENV variables if set.
func initConfig(cfg *signr.Signr) func() {
	return func() {
		viper.SetConfigName(signr.ConfigName)
		viper.SetConfigType(signr.ConfigExt)
		viper.AddConfigPath(cfg.DataDir)
		// read in environment variables that match
		viper.SetEnvPrefix(signr.AppName)
		viper.AutomaticEnv()
		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err == nil && cfg.Verbose {
			cfg.Log("Using config file: %s\n", viper.ConfigFileUsed())
		}

		// if pass is given on CLI it overrides environment, but if it is empty and environment has a value, load it
		if Pass == "" {
			if p := viper.GetString("pass"); p != "" {
				Pass = p
			}
		}
	}
}
