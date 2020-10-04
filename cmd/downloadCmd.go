package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"
	"github.com/tortlewortle/go_twitch_vod/pkg/tvod"
	"io"
	"os"
	"path"
	"runtime"
	"sync"
)

var downloadCmd = &cobra.Command{
	Use:   "download <video_id> <quality>",
	Short: "Get vod info",
	Args:  cobra.RangeArgs(1, 2),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		tmpdir := cmd.Flag("tmpdir").Value.String()

		if err := os.MkdirAll(path.Join(tmpdir, args[0]), 0766); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return err
			}
		}
		return nil
	},
	PostRunE: func(cmd *cobra.Command, args []string) error {
		tmpdir := cmd.Flag("tmpdir").Value.String()
		if err := os.RemoveAll(tmpdir); err != nil {
			return fmt.Errorf("could not cleanup tmp directory: %w", err)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		selectedSource := "source"
		if len(args) == 2 {
			selectedSource = args[1]
		}
		filename := cmd.Flag("out").Value.String()
		if filename == "" {
			filename = fmt.Sprintf("%s_%s.ts", args[0], selectedSource)
		}

		vod := tvod.NewVod(args[0])
		err := vod.Load(cmd.Context())
		if err != nil {
			return err
		}
		var sourceToDownload tvod.VodSource
		for _, source := range vod.Sources {
			desiredSource := selectedSource
			if selectedSource == "source" {
				desiredSource = "chunked"
			}
			if source.Name == desiredSource {
				sourceToDownload = source
			}
		}
		if sourceToDownload.Name == "" {
			return fmt.Errorf("source: %s does not exist", selectedSource)
		}
		fmt.Printf("Downloading %s:%s\n", args[0], sourceToDownload.Name)

		tmpdir := cmd.Flag("tmpdir").Value.String()

		parts, err := sourceToDownload.GetParts(cmd.Context())
		if err != nil {
			return err
		}
		dir := path.Join(tmpdir, args[0])
		concurrent := cmd.Flag("concurrent").Value.String()
		if concurrent == "true" {
			err = downloadVodConcurrently(cmd.Context(), dir, parts)
			if err != nil {
				return err
			}
		} else {
			err = downloadVod(cmd.Context(), dir, parts)
			if err != nil {
				return err
			}
		}
		err = mergeParts(filename, dir, parts)
		if err != nil {
			return err
		}
		return nil
	},
}

func mergeParts(filename string, tmpdir string, parts []tvod.VodPart) error {
	merged, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer merged.Close()

	for _, p := range parts {
		f, err := os.OpenFile(path.Join(tmpdir, p.Name), os.O_RDONLY, 0666)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(merged, f)
		if err != nil {
			return fmt.Errorf("could not copy: %w", err)
		}
		f.Close()
	}
	return nil
}

func downloadVod(ctx context.Context, dir string, parts []tvod.VodPart) error {
	bar := pb.StartNew(len(parts))
	defer bar.Finish()
	for _, part := range parts {
		err := downloadFile(ctx, path.Join(dir, part.Name), part)
		if err != nil {
			return err
		}
		bar.Increment()
	}
	return nil
}

func downloadFile(ctx context.Context, filename string, part tvod.VodPart) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = part.Download(ctx, f)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	downloadCmd.Flags().String("tmpdir", "./tmp", "tmp dir")
	downloadCmd.Flags().StringP("out", "o", "", "output file")
	downloadCmd.Flags().BoolP("concurrent", "c", false, "multi-thread download (experimental)")
}

func downloadVodConcurrently(ctx context.Context, dir string, parts []tvod.VodPart) error {
	ctx, cancel := context.WithCancel(context.TODO())

	partChan := make(chan tvod.VodPart, len(parts))
	errChan := make(chan error)

	bar := pb.StartNew(len(parts))
	defer bar.Finish()

	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		//fmt.Printf("Starting thread: %d\n", i)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case p := <-partChan:
					err := downloadFile(ctx, path.Join(dir, p.Name), p)
					wg.Done()
					if err != nil {
						errChan <- err
						return
					}
					bar.Increment()
				}
			}
		}()
	}
	for _, part := range parts {
		//fmt.Printf("Sending in: %s\n", part.Name)
		wg.Add(1)
		partChan <- part
	}
	go func() {
		wg.Wait()
		cancel()
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errChan:
		fmt.Println(err)
		close(errChan)
		cancel()
		return err
	}
}
