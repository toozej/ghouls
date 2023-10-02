package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/toozej/golang-starter/internal/math"
	"github.com/toozej/golang-starter/pkg/config"
	"github.com/toozej/golang-starter/pkg/man"
	"github.com/toozej/golang-starter/pkg/version"
)

func main() {
	// load application configurations
	if err := config.LoadConfig("./config"); err != nil {
		panic(fmt.Errorf("invalid application configuration: %s", err))
	}

	command := &cobra.Command{
		Use:   "golang-starter",
		Short: "golang starter examples",
		Long:  `Examples of using math library, cobra and viper modules in golang`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(config.Config.ConfigVar)

			addMessage := math.Add(1, 2)
			fmt.Println(addMessage)

			subMessage := math.Subtract(2, 2)
			fmt.Println(subMessage)
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
