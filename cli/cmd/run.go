/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/sath-run/engine/cli/request"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run jobs",
	Long:  `Run sath engine, it will ask jobs from remote sath server and execute them`,
	Run:   runEngine,
}

func runEngine(cmd *cobra.Command, args []string) {
	res, code := request.SendRequestToEngine(http.MethodPost, "/services/start", nil)
	if code == http.StatusUnauthorized {
		fmt.Println("login is required to run sath engine")
		return
	} else if code >= 400 {
		log.Fatal(res, code)
	}
	fmt.Println("successfully run sath-engine")
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
