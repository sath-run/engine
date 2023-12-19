/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/sath-run/engine/cli/request"
	"github.com/sath-run/engine/constants"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get sath engine status",
	Long:  `Get sath engine status`,
	Run:   runStatus,
}

func printStatusResult(result map[string]interface{}) {
	var status string = result["status"].(string)
	switch status {
	case "WAITING":
		fmt.Println("sath-engine is waiting")
		fmt.Println("  use `sath run` to run jobs")
	case "UNINITIALIZED", "STARTING":
		fmt.Println("sath-engine is starting, it may take a few seconds")
	case "RUNNING":
		fmt.Println("sath-engine is running")
		if jobs, ok := result["jobs"].([]interface{}); ok {
			fmt.Println("Current running jobs:")
			fmt.Println("*****************************************")
			for _, j := range jobs {
				if job, ok := j.(map[string]any); ok {
					fmt.Println("  ", job["execId"])
				}
			}
			fmt.Println("*****************************************")
			fmt.Println("  use `sath jobs` to view detail of jobs")
		} else {
			fmt.Println("No job is accpeted yet")
		}
	default:
		fmt.Println("unknown status, please contact service@sath.run for support")
	}
}

func runStatus(cmd *cobra.Command, args []string) {
	fmt.Println("sath version:", constants.Version)
	user := request.EngineGet("/users/info")
	if email, ok := user["email"].(string); ok && len(email) > 0 {
		fmt.Println("*****************************************")
		fmt.Println("sath-engine is logged in by user:")
		fmt.Println("email:", email)
		fmt.Println("name: ", user["name"])
		fmt.Println("*****************************************")
	} else {
		fmt.Println("no user is logged in")
		fmt.Println("  use `sath login` to login your account")
	}
	status := request.EngineGet("/services/status")
	printStatusResult(status)
}

func init() {
	rootCmd.AddCommand(statusCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// statusCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
}
