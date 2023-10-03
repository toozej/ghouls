package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/toozej/ghouls/internal/ghouls"
	"github.com/toozej/ghouls/pkg/man"
	"github.com/toozej/ghouls/pkg/version"
)

func main() {
	command := &cobra.Command{
		Use:   "ghouls",
		Short: "Go URL Bookmarking Service",
		Long:  `Go URL Bookmarking Service`,
		Run: func(cmd *cobra.Command, args []string) {
			ghouls.Serve()

		},
	}

	command.AddCommand(
		man.NewManCmd(),
		version.Command(),
	)

	if err := command.Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

}
