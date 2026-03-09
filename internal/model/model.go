package model

import "time"

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
)

type Job struct {
	ID          string    `json:"id"`
	Status      JobStatus `json:"status"`
	Operation   string    `json:"operation"`
	InputName   string    `json:"input_name"`
	OutputName  string    `json:"output_name,omitempty"`
	OutputURL   string    `json:"output_url,omitempty"`
	Width       int       `json:"width,omitempty"`
	Height      int       `json:"height,omitempty"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

type ResizeParams struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Fit    string `json:"fit"` // "fill", "contain", "cover"
}

type CropParams struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type RotateParams struct {
	Angle float64 `json:"angle"` // 90, 180, 270
}

type ConvertParams struct {
	Format  string `json:"format"` // "jpeg", "png", "gif", "webp"
	Quality int    `json:"quality"`
}

type FilterParams struct {
	Type      string  `json:"type"` // "grayscale", "blur", "sharpen", "brightness", "contrast"
	Intensity float64 `json:"intensity"`
}

type StatsResponse struct {
	TotalJobs     int            `json:"total_jobs"`
	ByStatus      map[string]int `json:"by_status"`
	ByOperation   map[string]int `json:"by_operation"`
	StoredImages  int            `json:"stored_images"`
	StorageBytes  int64          `json:"storage_bytes"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
}
