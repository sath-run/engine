/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/sath-run/engine/cli/request"
	"github.com/spf13/cobra"
)

// shutdownCmd represents the shutdown command
var shutdownCmd = &cobra.Command{
	Use:   "shutdown",
	Short: "Shutdown Sath engine",
	Long: `Shutdown Sath engine.
After shutdownping, Sath engine will cancel current running job
and will no longer start new jobs`,
	Run: runShutdown,
}

func runShutdown(cmd *cobra.Command, args []string) {
	resp := request.EnginePost("/services/stop", map[string]interface{}{"wait": true})
	fmt.Println(resp["message"])
	pid, err := request.FindRunningDaemonPid()
	if err != nil {
		fmt.Println(err)
		return
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("cannot find the process of pid", pid)
		return
	}
	if err := process.Kill(); err != nil {
		fmt.Printf("fail to kill process: %+v", err)
		return
	}
	fmt.Println("Sath engine successfully shutdown")
}

func init() {
	rootCmd.AddCommand(shutdownCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// shutdownCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// shutdownCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	shutdownCmd.Flags().BoolP("wait", "w", false, "Wait for job completion before exit")
}
