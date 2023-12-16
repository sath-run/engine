/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/sath-run/engine/cli/request"
	"github.com/spf13/cobra"
)

// jobsCmd represents the jobs command
var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "List jobs",
	Long:  `List SATH engine jobs`,
	Run:   runJobs,
}

type JobStatusResult struct {
	Jobs []struct {
		Id          string  `json:"id"`
		Message     string  `json:"message"`
		Status      string  `json:"status"`
		Progress    float64 `json:"progress"`
		CreatedAt   int64   `json:"createdAt"`
		CompletedAt int64   `json:"completedAt"`
		ContainerId string  `json:"containerId"`
		Image       string  `json:"image"`
	} `json:"jobs"`
}

func fmtDuration(d time.Duration) string {
	if d > time.Hour {
		amount := math.Round(d.Hours())
		if amount == 1 {
			return strconv.Itoa(int(amount)) + " hour"
		} else {
			return strconv.Itoa(int(amount)) + " hours"
		}
	} else if d > time.Minute {
		amount := math.Round(d.Minutes())
		if amount == 1 {
			return strconv.Itoa(int(amount)) + " minute"
		} else {
			return strconv.Itoa(int(amount)) + " minutes"
		}
	} else {
		amount := math.Round(d.Seconds())
		if amount == 1 {
			return strconv.Itoa(int(amount)) + " second"
		} else {
			return strconv.Itoa(int(amount)) + " seconds"
		}
	}
}

func runJobs(cmd *cobra.Command, args []string) {
	follow, err := cmd.Flags().GetBool("follow")
	if err != nil {
		log.Fatal(err)
	}
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		log.Fatal(err)
	}
	if follow {
		if all {
			fmt.Println("[WARNING] `--all` does not apply to `--follow` mode")
		}
		resp, err := http.Get(request.Origin + "/jobs/stream")
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			response := scanner.Text()
			if strings.HasPrefix(response, "data:") {
				response = strings.TrimPrefix(response, "data:")
				var result = JobStatusResult{}
				if err := json.Unmarshal([]byte(response), &result); err != nil {
					log.Fatal(err)
				}
				printJobs(result)
				fmt.Println()
			}
		}
	} else {
		url := "/jobs"
		if all {
			url += "?filter=all"
		}
		response := request.EngineGet(url)
		var result JobStatusResult
		if mapstructure.Decode(&response, &result); err != nil {
			log.Fatal(err)
		}
		printJobs(result)
	}
}

func printJobs(result JobStatusResult) {
	fmt.Printf("%-10s %-14s %-10s %-30s %-16s %-16s %-16s\n",
		"JOB ID", "STATUS", "PROGRESS", "IMAGE", "CONTAINER ID", "CREATED", "COMPLETED")
	for _, job := range result.Jobs {
		jobId := job.Id
		if len(jobId) > 10 {
			jobId = jobId[2:10]
		}
		createdAt := time.Unix(job.CreatedAt, 0)
		completedAt := time.Unix(job.CompletedAt, 0)
		image := strings.Split(job.Image, "@")[0]
		if len(image) > 28 {
			image = image[:25] + "..."
		}
		created := fmtDuration(time.Since(createdAt)) + " ago"
		completed := ""
		if !completedAt.IsZero() {
			completed = fmtDuration(time.Since(completedAt)) + " ago"
		}
		containerId := job.ContainerId
		if len(containerId) > 12 {
			containerId = containerId[:12]
		}
		fmt.Printf("%-10s %-14s %-10s %-30s %-16s %-16s %-16s\n",
			jobId, job.Status,
			fmt.Sprintf("%.2f%%", job.Progress),
			image, containerId,
			created,
			completed,
		)
	}
}

func init() {
	rootCmd.AddCommand(jobsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// jobsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// jobsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	jobsCmd.Flags().BoolP("follow", "f", false, "This option cause sath to stream status")
	jobsCmd.Flags().BoolP("all", "a", false, "Show all the jobs including finished ones")
}
