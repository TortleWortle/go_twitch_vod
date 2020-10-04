package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/tortlewortle/go_twitch_vod/pkg/tvod"
)

var infoCmd = &cobra.Command{
	Use:   "info <video_id>",
	Short: "Get vod info",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vod := tvod.NewVod(args[0])
		err := vod.Load(cmd.Context())
		if err != nil {
			return err
		}

		fmt.Println("Quality:")
		for _, source := range vod.Sources {
			name := source.Name
			if source.Name == "chunked" {
				name = "source"
			}
			fmt.Printf("- %s\n", name)
		}
		return nil
	},
}
