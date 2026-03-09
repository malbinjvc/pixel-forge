package handler

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/malbinjvc/pixel-forge/internal/model"
	"github.com/malbinjvc/pixel-forge/internal/storage"
)

func setupTest(t *testing.T) *Handler {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	return New(store)
}

func testPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 100, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.NRGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func uploadImage(t *testing.T, h *Handler) string {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.png")
	part.Write(testPNG())
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	h.Upload(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("upload returned %d: %s", w.Code, w.Body.String())
	}

	var job model.Job
	json.NewDecoder(w.Body).Decode(&job)
	return job.ID
}

func TestHealth(t *testing.T) {
	h := setupTest(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp model.HealthResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %s", resp.Status)
	}
}

func TestUpload(t *testing.T) {
	h := setupTest(t)
	id := uploadImage(t, h)
	if id == "" {
		t.Error("expected non-empty job ID")
	}
}

func TestUploadMissingFile(t *testing.T) {
	h := setupTest(t)
	req := httptest.NewRequest(http.MethodPost, "/api/upload", strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	w := httptest.NewRecorder()

	h.Upload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetJob(t *testing.T) {
	h := setupTest(t)
	id := uploadImage(t, h)

	req := httptest.NewRequest(http.MethodGet, "/api/jobs/"+id, nil)
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()

	h.GetJob(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetJobNotFound(t *testing.T) {
	h := setupTest(t)
	req := httptest.NewRequest(http.MethodGet, "/api/jobs/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	h.GetJob(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestListJobs(t *testing.T) {
	h := setupTest(t)
	uploadImage(t, h)

	req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
	w := httptest.NewRecorder()

	h.ListJobs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var jobs []model.Job
	json.NewDecoder(w.Body).Decode(&jobs)
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
}

func TestGetImage(t *testing.T) {
	h := setupTest(t)
	id := uploadImage(t, h)

	req := httptest.NewRequest(http.MethodGet, "/api/images/"+id, nil)
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()

	h.GetImage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		t.Errorf("expected image content type, got %s", ct)
	}
}

func TestGetImageNotFound(t *testing.T) {
	h := setupTest(t)
	req := httptest.NewRequest(http.MethodGet, "/api/images/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	h.GetImage(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestProcessResize(t *testing.T) {
	h := setupTest(t)
	id := uploadImage(t, h)

	params := model.ResizeParams{Width: 50, Height: 40}
	body, _ := json.Marshal(params)

	req := httptest.NewRequest(http.MethodPost, "/api/images/"+id+"/resize", bytes.NewReader(body))
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()

	h.ProcessResize(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var job model.Job
	json.NewDecoder(w.Body).Decode(&job)
	if job.Status != model.StatusCompleted {
		t.Errorf("expected completed, got %s", job.Status)
	}
}

func TestProcessCrop(t *testing.T) {
	h := setupTest(t)
	id := uploadImage(t, h)

	params := model.CropParams{X: 10, Y: 10, Width: 50, Height: 40}
	body, _ := json.Marshal(params)

	req := httptest.NewRequest(http.MethodPost, "/api/images/"+id+"/crop", bytes.NewReader(body))
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()

	h.ProcessCrop(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProcessRotate(t *testing.T) {
	h := setupTest(t)
	id := uploadImage(t, h)

	params := model.RotateParams{Angle: 90}
	body, _ := json.Marshal(params)

	req := httptest.NewRequest(http.MethodPost, "/api/images/"+id+"/rotate", bytes.NewReader(body))
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()

	h.ProcessRotate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProcessFilter(t *testing.T) {
	h := setupTest(t)
	id := uploadImage(t, h)

	params := model.FilterParams{Type: "grayscale"}
	body, _ := json.Marshal(params)

	req := httptest.NewRequest(http.MethodPost, "/api/images/"+id+"/filter", bytes.NewReader(body))
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()

	h.ProcessFilter(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProcessConvert(t *testing.T) {
	h := setupTest(t)
	id := uploadImage(t, h)

	params := model.ConvertParams{Format: "jpeg", Quality: 80}
	body, _ := json.Marshal(params)

	req := httptest.NewRequest(http.MethodPost, "/api/images/"+id+"/convert", bytes.NewReader(body))
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()

	h.ProcessConvert(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify we can download the converted image
	var job model.Job
	json.NewDecoder(w.Body).Decode(&job)

	imgReq := httptest.NewRequest(http.MethodGet, "/api/images/"+job.ID, nil)
	imgReq.SetPathValue("id", job.ID)
	imgW := httptest.NewRecorder()

	h.GetImage(imgW, imgReq)
	if imgW.Code != http.StatusOK {
		t.Errorf("expected 200 for converted image, got %d", imgW.Code)
	}
}

func TestProcessResizeNotFound(t *testing.T) {
	h := setupTest(t)

	params := model.ResizeParams{Width: 50, Height: 40}
	body, _ := json.Marshal(params)

	req := httptest.NewRequest(http.MethodPost, "/api/images/nonexistent/resize", bytes.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	h.ProcessResize(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestStats(t *testing.T) {
	h := setupTest(t)
	uploadImage(t, h)

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	w := httptest.NewRecorder()

	h.Stats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var stats model.StatsResponse
	json.NewDecoder(w.Body).Decode(&stats)
	if stats.TotalJobs != 1 {
		t.Errorf("expected 1 total job, got %d", stats.TotalJobs)
	}
	if stats.StoredImages != 1 {
		t.Errorf("expected 1 stored image, got %d", stats.StoredImages)
	}
}

func TestProcessInvalidJSON(t *testing.T) {
	h := setupTest(t)
	id := uploadImage(t, h)

	req := httptest.NewRequest(http.MethodPost, "/api/images/"+id+"/resize", strings.NewReader("not json"))
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()

	h.ProcessResize(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// Suppress unused import warning
var _ = io.Discard
