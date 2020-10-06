package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var (
	cliVersion = "0.0.1"
	rootCmd = &cobra.Command{
		Use: "download",
		Short: "a cli download tool",
		Version: cliVersion,
		Run: func(cmd *cobra.Command, args []string) {
			showVersion()
		},
	}
)


func init(){
	rootCmd.Flags().BoolP("version", "v", false, "version of bear")
}

func Execute(){
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func showVersion(){
	fmt.Printf("bear version %s\n", cliVersion)
}
