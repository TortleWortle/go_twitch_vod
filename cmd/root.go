package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "tvod",
	Short: "Download twitch vods.",
	Long:  `Tool for downloading twitch vods.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Do Stuff Here
		return cmd.Help()
	},
}

func Execute() {
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(downloadCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
