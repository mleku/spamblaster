package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"mleku.online/git/signr/pkg/signr"
	"os"
)

var (
	Pass string
	s    *signr.Signr
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "notegen",
	Short: "generate and random generated send event notes to a relay",
	Long: `notegen

random note generator for testing purposes.

fills in content with a random selection from the devlorem quote list, optionally  adds a reply reference.`,
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

	var e error
	// because this is a CLI app we know the user can enter passwords this way.
	// other types of apps using this can load the environment variables.
	s, e = signr.Init(signr.PasswordEntryViaTTY)
	if e != nil {
		s.Fatal("fatal error: %s\n", e)
	}
	rootCmd.PersistentFlags().StringVarP(&Pass,
		"pass", "p", "", "password, if needed, to unlock signing key")
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
