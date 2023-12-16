/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

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

func findRunningDaemonPid() (pid int, err error) {
	command := exec.Command("bash", "-c", "ps | grep sath-engine | grep -v grep")
	res, err := command.Output()
	if err != nil {
		err = errors.New("cannot find the running pid of sath")
		return
	}
	pid, err = strconv.Atoi(strings.Fields(string(res))[0])
	if err != nil {
		return
	}
	return
}

func runShutdown(cmd *cobra.Command, args []string) {
	// wait, err := cmd.Flags().GetBool("wait")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// resp := EnginePost("/services/shutdown", map[string]interface{}{"wait": wait})
	// fmt.Println(resp["message"])
	pid, err := findRunningDaemonPid()
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
