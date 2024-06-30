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

func startEngineBySystemctl() bool {
	systemctl, err := exec.LookPath("systemd")
	if err != nil || len(systemctl) == 0 {
		return false
	}
	fmt.Println("trying to start sath by systemctl")
	cmd := exec.Command("systemctl", "start", "sath")
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("fail to run systemctl: ", err)
		return false
	}
	fmt.Println(string(out))
	return true
}

func startEngine() {
	fmt.Println("starting sath engine")
	var buf bytes.Buffer

	if startEngineBySystemctl() {
		// nothing to do
	} else {
		fmt.Println("starting sath by executing sath-engine binary")
		// start engine by executing sath-engine
		command := exec.Command(filepath.Join(utils.ExecutableDir, "sath-engine"))
		command.Stderr = &buf
		err := command.Start()
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Second)
	}

	if ok := request.PingSathEngine(); ok {
		fmt.Println("sath engine successfully started")
	} else {
		if pid, _ := request.FindRunningDaemonPid(); pid != 0 {
			log.Fatalf("fail to ping sath engine: %s", buf.String())
		} else {
			log.Fatalf("fail to start sath engine: %s", buf.String())
		}
	}
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
	if pid, _ := request.FindRunningDaemonPid(); pid != 0 {
		fmt.Println("Sath engine is running")
		if !request.PingSathEngine() {
			log.Fatalf("fail to ping sath engine")
		}
	} else {
		if ok, version := checkIfUpgradeNeeded(); ok {
			if err := upgradeExecutables(); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("your sath has been successfully upgraded to version %s\n", version)
			return
		} else {
			startEngine()
		}
	}
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
