/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/sath-run/engine/cli/request"
	"github.com/sath-run/engine/constants"
	"github.com/sath-run/engine/utils"
	"github.com/spf13/cobra"
)

// upgradeCmd represents the upgrade command
var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: runUpgrade,
}

func checkSathLatestVersion() (string, error) {
	res, err := http.Get(fmt.Sprintf("https://download.sath.run/binaries/%s/%s/VERSION", runtime.GOOS, runtime.GOARCH))
	if err != nil {
		return "", fmt.Errorf("fail to get latest version info, %+v", err)
	}
	defer res.Body.Close()
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("fail to read version request body, %+v", err)

	}
	return string(bytes), nil
}

func upgradeExecutables() error {
	var err error
	fmt.Println("downloading sath-engine")
	url := fmt.Sprintf("https://download.sath.run/binaries/%s/%s/sath-engine", runtime.GOOS, runtime.GOARCH)
	err = request.DownloadFile(filepath.Join(utils.ExecutableDir, "sath-engine"), url)
	if err != nil {
		return fmt.Errorf("fail to download sath-engine %+v", err)
	}

	fmt.Println("downloading sath-cli")
	url = fmt.Sprintf("https://download.sath.run/binaries/%s/%s/sath", runtime.GOOS, runtime.GOARCH)
	err = request.DownloadFile(filepath.Join(utils.ExecutableDir, "sath"), url)
	if err != nil {
		return fmt.Errorf("fail to download sath-cli %+v", err)
	}
	return nil
}

func runUpgrade(cmd *cobra.Command, args []string) {
	if pid, _ := request.FindRunningDaemonPid(); pid != 0 {
		fmt.Println("Sath engine is still running, please use shutdown it first")
		return
	}
	fmt.Println("Checking the latest version")
	latest, err := checkSathLatestVersion()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("current version:", constants.Version)
	fmt.Println("latest available version:", latest)
	if constants.Version == latest {
		fmt.Println("your sath version is already the latest")
		return
	}
	if err := upgradeExecutables(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("your sath version is successfully upgraded to %s\n", latest)

}

func init() {
	rootCmd.AddCommand(upgradeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// upgradeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// upgradeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
