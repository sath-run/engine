/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/sath-run/engine/cli/request"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login",
	Long:  `Login your SATH account, so that your contribution for running jobs will be scored to your account`,
	Run:   runLogin,
}

func runLogin(cmd *cobra.Command, args []string) {
	request.TryPing()
	reader := bufio.NewReader(os.Stdin)
	var err error
	username, err := cmd.Flags().GetString("username")
	if err != nil {
		log.Fatal(err)
	}
	password, err := cmd.Flags().GetString("password")
	if err != nil {
		log.Fatal(err)
	}
	for len(username) == 0 {
		fmt.Print("Enter Username: ")
		username, err = reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		} else {
			username = strings.Trim(username, "\n")
		}
	}
	for len(password) == 0 {
		fmt.Print("Enter Password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatal(err)
		}

		password = string(bytePassword)
		fmt.Println("")
	}
	res, status := request.SendRequestToEngine(http.MethodPost, "/users/login", map[string]interface{}{
		"Username": username,
		"Password": password,
	})
	if status == http.StatusOK {
		fmt.Println("login success")
	} else {
		fmt.Printf("login failed: ")
		if message, ok := res["message"]; ok {
			if str, ok := message.(string); ok {
				fmt.Println(str)
				return
			}
		}
		fmt.Println()
	}
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loginCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loginCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	loginCmd.Flags().StringP("username", "u", "", "Username")
	loginCmd.Flags().StringP("password", "p", "", "Password")
}
