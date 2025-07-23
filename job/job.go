package job

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Priority int

const (
	PriorityLow    Priority = 1
	PriorityNormal Priority = 2
	PriorityHigh   Priority = 3
	PriorityUrgent Priority = 4
)

type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	Priority    Priority               `json:"priority"`
	Status      Status                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	WorkerID    *string                `json:"worker_id,omitempty"`
	Attempts    int                    `json:"attempts"`
	MaxAttempts int                    `json:"max_attempts"`
	Error       *string                `json:"error,omitempty"`
	Result      *Result                `json:"result,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type Result struct {
	Data    interface{}            `json:"data"`
	Metrics map[string]interface{} `json:"metrics"`
}

func New(jobType string, payload map[string]interface{}, priority Priority) *Job {
	now := time.Now()
	return &Job{
		ID:          uuid.New().String(),
		Type:        jobType,
		Payload:     payload,
		Priority:    priority,
		Status:      StatusPending,
		CreatedAt:   now,
		Attempts:    0,
		MaxAttempts: 3,
		Metadata:    make(map[string]interface{}),
	}
}

func (j *Job) Start(workerID string) error {
	if j.Status != StatusPending {
		return fmt.Errorf("job %s cannot be started, current status: %s", j.ID, j.Status)
	}

	now := time.Now()
	j.Status = StatusRunning
	j.StartedAt = &now
	j.WorkerID = &workerID
	j.Attempts++

	return nil
}

func (j *Job) Complete(result *Result) error {
	if j.Status != StatusRunning {
		return fmt.Errorf("job %s cannot be completed, current status: %s", j.ID, j.Status)
	}

	now := time.Now()
	j.Status = StatusCompleted
	j.CompletedAt = &now
	j.Result = result

	return nil
}

func (j *Job) Fail(err error) error {
	if j.Status != StatusRunning {
		return fmt.Errorf("job %s cannot be failed, current status: %s", j.ID, j.Status)
	}

	now := time.Now()
	errMsg := err.Error()

	if j.Attempts >= j.MaxAttempts {
		j.Status = StatusFailed
		j.CompletedAt = &now
	} else {
		j.Status = StatusPending
	}

	j.Error = &errMsg
	j.WorkerID = nil

	return nil
}

func (j *Job) Cancel() error {
	if j.Status == StatusCompleted || j.Status == StatusFailed {
		return fmt.Errorf("job %s cannot be cancelled, current status: %s", j.ID, j.Status)
	}

	now := time.Now()
	j.Status = StatusCancelled
	j.CompletedAt = &now
	j.WorkerID = nil

	return nil
}

func (j *Job) CanRetry() bool {
	return j.Status == StatusFailed && j.Attempts < j.MaxAttempts
}

func (j *Job) Duration() time.Duration {
	if j.StartedAt == nil {
		return 0
	}

	endTime := time.Now()
	if j.CompletedAt != nil {
		endTime = *j.CompletedAt
	}

	return endTime.Sub(*j.StartedAt)
}

func (j *Job) ToJSON() ([]byte, error) {
	return json.Marshal(j)
}

func (j *Job) FromJSON(data []byte) error {
	return json.Unmarshal(data, j)
}

func (j *Job) SetMetadata(key string, value interface{}) {
	if j.Metadata == nil {
		j.Metadata = make(map[string]interface{})
	}
	j.Metadata[key] = value
}

func (j *Job) GetMetadata(key string) (interface{}, bool) {
	if j.Metadata == nil {
		return nil, false
	}
	value, exists := j.Metadata[key]
	return value, exists
}
