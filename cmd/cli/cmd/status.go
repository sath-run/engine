/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

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
	fmt.Println("SATH Version:", result["version"])
	var status string = result["status"].(string)
	switch status {
	case "WAITING":
		fmt.Println("SATH Engine is waiting")
		fmt.Println("  use: `sath start` to start it")
	case "UNINITIALIZED", "STARTING":
		fmt.Println("SATH Engine is starting, it may take a few seconds")
	case "RUNNING":
		fmt.Println("SATH Engine is running")
		if jobs, ok := result["jobs"].([]interface{}); ok {
			fmt.Println("Current running jobs:")
			fmt.Println("*****************************************")
			for _, j := range jobs {
				if job, ok := j.(map[string]any); ok {
					fmt.Println("  ", job["execId"])
				}
			}
			fmt.Println("*****************************************")
			fmt.Println("Use: `sath jobs` to view detail of jobs")
		} else {
			fmt.Println("No job is accpeted yet")
		}
	default:
		fmt.Println("unknown status, please contact service@sath.run for support")
	}
	// if result == nil {
	// 	fmt.Println("no job is running")
	// } else {
	// 	fmt.Printf("id: %s\n", result["id"])
	// 	fmt.Printf("status: %s\n", result["status"])
	// 	fmt.Printf("progress: %f\n", result["progress"])
	// 	fmt.Printf("message: %s\n", result["message"])
	// 	fmt.Println()
	// }
}

func runStatus(cmd *cobra.Command, args []string) {
	result := EngineGet("/services/status")
	printStatusResult(result)
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
