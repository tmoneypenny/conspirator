package cmd

import (
	"fmt"
	"net/http"
	"os"

	_ "net/http/pprof"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Project
const (
	ProjectName    = "conspirator"
	ProjectLicense = "apache2.0"
)

var rootCmd = &cobra.Command{
	Use:   ProjectName,
	Short: fmt.Sprintf("%s standalone server", ProjectName),
	Long: fmt.Sprintf(`%s is a standalone collaborator-like server
that contains some enhanced functionality useful in for testing.`, ProjectName),
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the standalone server",
	Long: fmt.Sprintf(`%s is a standalone collaborator-like server
that contains some enhanced functionality useful in for testing.`, ProjectName),
	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetBool("profile") {
			log.Debug().Msgf("Enabling Profiler: %v. See: pprof %s",
				viper.GetBool("profile"), "http://localhost:6060/debug/pprof/")
			go http.ListenAndServe("localhost:6060", nil)
		}
		log.Info().Msgf("Starting %s server...", ProjectName)
		serverHandler()
	},
}

// add command for profile
func init() {
	// global flags

	// cmd Flags
	startCmd.Flags().StringP("config", "c", "",
		fmt.Sprintf("config file (default is $HOME/%s/configs/%s.config)", ProjectName, ProjectName))
	startCmd.Flags().BoolP("profile", "p", false, "enable profiler")

	viper.BindPFlags(startCmd.Flags())

	// Add sub-commands
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(configCmd)

	// prevent init for root help cmd
	if rootCmd.Use == ProjectName {
		cobra.OnInitialize(initConfig, initLogger)
	}
}

func execute() error {
	return rootCmd.Execute()
}

// Execute initiates the server
func Execute() {
	if err := execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
