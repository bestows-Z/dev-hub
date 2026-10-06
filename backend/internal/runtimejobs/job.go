package runtimejobs

import "time"

type Job struct {
	ID          uint64     `json:"id" gorm:"primaryKey"`
	ProjectID   uint64     `json:"project_id"`
	Action      string     `json:"action"`
	Status      string     `json:"status"`
	ErrorText   string     `json:"error_text"`
	RequestedBy uint64     `json:"requested_by"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at"`
}

func (Job) TableName() string { return "project_runtime_jobs" }
