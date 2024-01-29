package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	_ "go.uber.org/automaxprocs"

	"github.com/toozej/ghouls/internal/ghouls"
	"github.com/toozej/ghouls/pkg/man"
	"github.com/toozej/ghouls/pkg/version"
)

var rootCmd = &cobra.Command{
	Use:              "ghouls",
	Short:            "Go URL Bookmarking Service",
	Long:             `Go URL Bookmarking Service`,
	PersistentPreRun: rootCmdPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		ghouls.Serve()
	},
}

func rootCmdPreRun(cmd *cobra.Command, args []string) {
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func init() {
	// create rootCmd-level flags
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug-level logging")

	rootCmd.AddCommand(
		man.NewManCmd(),
		version.Command(),
	)
}
