/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/sath-run/engine/pkg/utils"
	"github.com/spf13/cobra"
)

// startupCmd represents the startup command
var startupCmd = &cobra.Command{
	Use:   "startup",
	Short: "Startup Sath engine",
	Long: `Startup Sath engine,
after startuping, sath will automatically accept and run jobs`,
	Run: runStartup,
}

func runStartup(cmd *cobra.Command, args []string) {
	if pid, _ := findRunningDaemonPid(); pid != 0 {
		fmt.Println("Sath engine is already running")
		return
	}
	dir, err := utils.GetExecutableDir()
	if err != nil {
		panic(err)
	}
	command := exec.Command(filepath.Join(dir, "sath-daemon"))
	if err := command.Start(); err != nil {
		panic(err)
	}
	fmt.Println("Sath engine successfully started")
}

func init() {
	rootCmd.AddCommand(startupCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startupCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startupCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
