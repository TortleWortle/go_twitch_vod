package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/tortlewortle/go_twitch_vod/pkg/tvod"
)

var infoCmd = &cobra.Command{
	Use:   "info <video_id>",
	Short: "Get vod info",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vod := tvod.NewVod(args[0])
		slog.Info("Loading VOD")
		err := vod.Load(cmd.Context())
		if err != nil {
			return errors.Join(errors.New("loading vod"), err)
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
