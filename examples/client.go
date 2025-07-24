package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type JobResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type StatsResponse struct {
	JobsSubmitted int64  `json:"jobs_submitted"`
	JobsCompleted int64  `json:"jobs_completed"`
	JobsFailed    int64  `json:"jobs_failed"`
	WorkersActive int64  `json:"workers_active"`
	QueueSize     int64  `json:"queue_size"`
	Uptime        string `json:"uptime"`
}

type Client struct {
	baseURL string
	client  *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Health() (*HealthResponse, error) {
	resp, err := c.client.Get(c.baseURL + "/health")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, err
	}

	return &health, nil
}

func (c *Client) Stats() (*StatsResponse, error) {
	resp, err := c.client.Get(c.baseURL + "/stats")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

func (c *Client) SubmitJob(jobType, priority string) (*JobResponse, error) {
	url := fmt.Sprintf("%s/jobs?type=%s&priority=%s", c.baseURL, jobType, priority)

	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jobResp JobResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		return nil, err
	}

	return &jobResp, nil
}

func main() {
	client := NewClient("http://localhost:8080")

	fmt.Println("Go Task Processing System - Example Client")
	fmt.Println("==========================================")

	health, err := client.Health()
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		return
	}
	fmt.Printf("Health: %+v\n", health)

	stats, err := client.Stats()
	if err != nil {
		fmt.Printf("Stats failed: %v\n", err)
		return
	}
	fmt.Printf("Stats: %+v\n", stats)

	jobTypes := []string{"email", "report", "backup", "sync", "cleanup"}
	priorities := []string{"low", "normal", "high", "urgent"}

	fmt.Println("\nSubmitting jobs...")
	for i := 0; i < 5; i++ {
		jobType := jobTypes[i%len(jobTypes)]
		priority := priorities[i%len(priorities)]

		job, err := client.SubmitJob(jobType, priority)
		if err != nil {
			fmt.Printf("Failed to submit job: %v\n", err)
			continue
		}

		fmt.Printf("Submitted job: %+v\n", job)
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println("\nWaiting for jobs to complete...")
	time.Sleep(2 * time.Second)

	finalStats, err := client.Stats()
	if err != nil {
		fmt.Printf("Final stats failed: %v\n", err)
		return
	}
	fmt.Printf("Final stats: %+v\n", finalStats)
}
