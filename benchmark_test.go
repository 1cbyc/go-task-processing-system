package main

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func BenchmarkJobSubmission(b *testing.B) {
	client := &http.Client{Timeout: 30 * time.Second}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		url := fmt.Sprintf("http://localhost:8080/jobs?type=benchmark&priority=normal")
		resp, err := client.Post(url, "application/json", bytes.NewBuffer([]byte("{}")))
		if err != nil {
			b.Fatalf("Failed to submit job: %v", err)
		}
		resp.Body.Close()
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	client := &http.Client{Timeout: 30 * time.Second}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get("http://localhost:8080/health")
		if err != nil {
			b.Fatalf("Failed to get health: %v", err)
		}
		resp.Body.Close()
	}
}

func BenchmarkStats(b *testing.B) {
	client := &http.Client{Timeout: 30 * time.Second}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get("http://localhost:8080/stats")
		if err != nil {
			b.Fatalf("Failed to get stats: %v", err)
		}
		resp.Body.Close()
	}
}

func BenchmarkConcurrentJobSubmission(b *testing.B) {
	client := &http.Client{Timeout: 30 * time.Second}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			url := fmt.Sprintf("http://localhost:8080/jobs?type=concurrent&priority=normal")
			resp, err := client.Post(url, "application/json", bytes.NewBuffer([]byte("{}")))
			if err != nil {
				b.Fatalf("Failed to submit job: %v", err)
			}
			resp.Body.Close()
		}
	})
}
