package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Result struct {
	Successes []res `json:"successes"`
	Failures  []res `json:"failures"`

	mu sync.Mutex
}

type res struct {
	Error     string `json:"error,omitempty"`
	Host      string `json:"host,omitempty"`
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
	TimeTaken string `json:"time_taken,omitempty"`
	Output    string `json:"output,omitempty"`
}

func (r *Result) AddResult(start, end time.Time, host string, output []byte, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := res{
		Host:      host,
		StartTime: start.Format(time.RFC3339),
		EndTime:   end.Format(time.RFC3339),
		TimeTaken: fmt.Sprintf("%fs", start.Sub(end).Seconds()),
		Output:    string(output),
	}

	if err != nil {
		result.Error = err.Error()
		r.Failures = append(r.Failures, result)
		return
	}

	r.Successes = append(r.Successes, result)
}

func (r *Result) MarshalJSON() ([]byte, error) {
	return json.Marshal(r)
}
