package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "1pa",
	Short: "1pa is a command line interface to 1password",
	Long:  "something something something",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello Cobra!")
		// Do Stuff Here
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
