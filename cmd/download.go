package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/HaHadaxigua/Devourer/example"
)

var (
	downloadCmd = &cobra.Command{
		Use:     "download",
		Aliases: []string{"dl"},
		Run: func(cmd *cobra.Command, args []string) {
			downloadLink := args[0]

			downloader := example

			if err != nil {
				fmt.Println("Don't know how to download: ", text)
				return
			}

			return
		},
	}
)

func init(){
	rootCmd.AddCommand(downloadCmd)
}