package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/malbinjvc/pixel-forge/internal/model"
	"github.com/malbinjvc/pixel-forge/internal/processor"
	"github.com/malbinjvc/pixel-forge/internal/storage"
)

const maxUploadSize = 10 << 20 // 10 MB

var allowedTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
}

type Handler struct {
	store   *storage.FileStorage
	jobs    map[string]*model.Job
	mu      sync.RWMutex
	startAt time.Time
}

func New(store *storage.FileStorage) *Handler {
	return &Handler{
		store:   store,
		jobs:    make(map[string]*model.Job),
		startAt: time.Now(),
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	resp := model.HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
		Uptime:  time.Since(h.startAt).Truncate(time.Second).String(),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "file too large (max 10MB)")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'image' field")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxUploadSize))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	contentType := http.DetectContentType(data)
	if !allowedTypes[contentType] {
		writeError(w, http.StatusBadRequest, "unsupported image type: "+contentType)
		return
	}

	jobID := generateID()
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".png"
	}
	filename := jobID + ext

	if err := h.store.Save(filename, data); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	job := &model.Job{
		ID:        jobID,
		Status:    model.StatusCompleted,
		Operation: "upload",
		InputName: filename,
		CreatedAt: time.Now(),
	}
	h.saveJob(job)

	writeJSON(w, http.StatusCreated, job)
}

func (h *Handler) ProcessResize(w http.ResponseWriter, r *http.Request) {
	var params model.ResizeParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	imageID := r.PathValue("id")
	h.processImage(w, imageID, "resize", func(data []byte) ([]byte, error) {
		img, format, err := processor.Decode(data)
		if err != nil {
			return nil, err
		}
		result, err := processor.Resize(img, params)
		if err != nil {
			return nil, err
		}
		return processor.Encode(result, format, 90)
	})
}

func (h *Handler) ProcessCrop(w http.ResponseWriter, r *http.Request) {
	var params model.CropParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	imageID := r.PathValue("id")
	h.processImage(w, imageID, "crop", func(data []byte) ([]byte, error) {
		img, format, err := processor.Decode(data)
		if err != nil {
			return nil, err
		}
		result, err := processor.Crop(img, params)
		if err != nil {
			return nil, err
		}
		return processor.Encode(result, format, 90)
	})
}

func (h *Handler) ProcessRotate(w http.ResponseWriter, r *http.Request) {
	var params model.RotateParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	imageID := r.PathValue("id")
	h.processImage(w, imageID, "rotate", func(data []byte) ([]byte, error) {
		img, format, err := processor.Decode(data)
		if err != nil {
			return nil, err
		}
		result, err := processor.Rotate(img, params)
		if err != nil {
			return nil, err
		}
		return processor.Encode(result, format, 90)
	})
}

func (h *Handler) ProcessFilter(w http.ResponseWriter, r *http.Request) {
	var params model.FilterParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	imageID := r.PathValue("id")
	h.processImage(w, imageID, "filter:"+params.Type, func(data []byte) ([]byte, error) {
		img, format, err := processor.Decode(data)
		if err != nil {
			return nil, err
		}
		result, err := processor.ApplyFilter(img, params)
		if err != nil {
			return nil, err
		}
		return processor.Encode(result, format, 90)
	})
}

func (h *Handler) ProcessConvert(w http.ResponseWriter, r *http.Request) {
	var params model.ConvertParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	imageID := r.PathValue("id")
	h.processImage(w, imageID, "convert:"+params.Format, func(data []byte) ([]byte, error) {
		return processor.Convert(data, params)
	})
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	h.mu.RLock()
	job, exists := h.jobs[id]
	h.mu.RUnlock()

	if !exists {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	jobs := make([]*model.Job, 0, len(h.jobs))
	for _, j := range h.jobs {
		jobs = append(jobs, j)
	}
	h.mu.RUnlock()

	writeJSON(w, http.StatusOK, jobs)
}

func (h *Handler) GetImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	h.mu.RLock()
	job, exists := h.jobs[id]
	h.mu.RUnlock()

	if !exists {
		writeError(w, http.StatusNotFound, "image not found")
		return
	}

	filename := job.InputName
	if job.OutputName != "" {
		filename = job.OutputName
	}

	data, err := h.store.Load(filename)
	if err != nil {
		writeError(w, http.StatusNotFound, "image file not found")
		return
	}

	ct := http.DetectContentType(data)
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	byStatus := make(map[string]int)
	byOp := make(map[string]int)
	for _, j := range h.jobs {
		byStatus[string(j.Status)]++
		byOp[j.Operation]++
	}
	total := len(h.jobs)
	h.mu.RUnlock()

	imgCount, imgBytes := h.store.Stats()

	resp := model.StatsResponse{
		TotalJobs:    total,
		ByStatus:     byStatus,
		ByOperation:  byOp,
		StoredImages: imgCount,
		StorageBytes: imgBytes,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) processImage(w http.ResponseWriter, imageID, operation string, fn func([]byte) ([]byte, error)) {
	if imageID == "" {
		writeError(w, http.StatusBadRequest, "missing image ID")
		return
	}

	// Find the source file
	var sourceFile string
	h.mu.RLock()
	for _, j := range h.jobs {
		if j.ID == imageID {
			if j.OutputName != "" {
				sourceFile = j.OutputName
			} else {
				sourceFile = j.InputName
			}
			break
		}
	}
	h.mu.RUnlock()

	if sourceFile == "" {
		writeError(w, http.StatusNotFound, "source image not found")
		return
	}

	data, err := h.store.Load(sourceFile)
	if err != nil {
		writeError(w, http.StatusNotFound, "image file not found")
		return
	}

	jobID := generateID()
	job := &model.Job{
		ID:        jobID,
		Status:    model.StatusProcessing,
		Operation: operation,
		InputName: sourceFile,
		CreatedAt: time.Now(),
	}
	h.saveJob(job)

	result, err := fn(data)
	if err != nil {
		job.Status = model.StatusFailed
		job.Error = err.Error()
		job.CompletedAt = time.Now()
		h.saveJob(job)
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	ext := filepath.Ext(sourceFile)
	if strings.Contains(operation, "convert:") {
		format := strings.TrimPrefix(operation, "convert:")
		ext = "." + format
	}
	outputName := jobID + ext
	if err := h.store.Save(outputName, result); err != nil {
		job.Status = model.StatusFailed
		job.Error = "failed to save output"
		h.saveJob(job)
		writeError(w, http.StatusInternalServerError, "failed to save output")
		return
	}

	job.Status = model.StatusCompleted
	job.OutputName = outputName
	job.OutputURL = "/api/images/" + jobID
	job.CompletedAt = time.Now()
	h.saveJob(job)

	writeJSON(w, http.StatusOK, job)
}

func (h *Handler) saveJob(job *model.Job) {
	h.mu.Lock()
	h.jobs[job.ID] = job
	h.mu.Unlock()
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, model.ErrorResponse{Error: msg, Code: status})
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
