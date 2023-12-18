/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/sath-run/engine/cli/request"
	"github.com/sath-run/engine/constants"
	"github.com/sath-run/engine/utils"
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

func startEngine() {
	fmt.Println("starting sath engine")
	var buf bytes.Buffer
	command := exec.Command(filepath.Join(utils.ExecutableDir, "sath-engine"))
	command.Stderr = &buf
	err := command.Start()

	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
	if ok := testSathEnginePing(); ok {
		log.Println("sath engine successfully started")
	} else {
		if pid, _ := findRunningDaemonPid(); pid != 0 {
			log.Fatalf("fail to ping sath engine: %s", buf.String())
		} else {
			log.Fatalf("fail to start sath engine: %s", buf.String())
		}
	}
}

func testSathEnginePing() bool {
	for i := 0; i < 3; i++ {
		time.Sleep(time.Second)
		// ping sath-engine to make sure it is started
		if request.Ping() {
			return true
		} else {
			continue
		}
	}

	return false
}

func checkIfUpgradeNeeded() (bool, string) {
	version, err := checkSathLatestVersion()
	if err != nil {
		// if fail to get the latest version, just stop here
		return false, version
	}
	if version != constants.Version {
		prompt := promptui.Prompt{
			Label:     fmt.Sprintf("A new version of sath (%s) was detected, upgrade now?", version),
			IsConfirm: true,
		}
		var result string
		for i := 0; i < 3; i++ {
			result, _ = prompt.Run()
			result = strings.TrimSpace(result)
			result = strings.ToLower(result)
			if result == "y" || result == "n" {
				break
			}
		}
		if result == "y" {
			return true, version
		}
	}
	return false, version
}

func runStartup(cmd *cobra.Command, args []string) {
	if pid, _ := findRunningDaemonPid(); pid != 0 {
		fmt.Println("Sath engine is running")
		if !testSathEnginePing() {
			log.Fatalf("fail to ping sath engine")
		}
	} else {
		if ok, version := checkIfUpgradeNeeded(); ok {
			if err := upgradeExecutables(); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("your sath version is successfully upgraded to %s\n", version)
			return
		} else {
			startEngine()
		}
	}
	request.EnginePost("/services/start", nil)
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
