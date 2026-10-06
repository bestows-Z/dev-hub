package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	"github.com/bestows-Z/dev-hub/backend/internal/gallery"
	"github.com/bestows-Z/dev-hub/backend/internal/storage"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestGalleryUploadPublishAndDeleteIntegration(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_GALLERY") != "1" {
		t.Skip("set DEVHUB_TEST_GALLERY=1 to test PostgreSQL and MinIO")
	}
	cfg := config.PostgresConfig{Host: "127.0.0.1", Port: 5432, User: os.Getenv("POSTGRES_USER"), Password: os.Getenv("POSTGRES_PASSWORD"), DBName: os.Getenv("POSTGRES_DB"), SSLMode: "disable", TimeZone: "UTC"}
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	bucket := os.Getenv("MINIO_BUCKET")
	if bucket == "" {
		bucket = "devhub-projects"
	}
	store, err := storage.New(config.StorageConfig{Endpoint: "127.0.0.1:9000", AccessKey: os.Getenv("MINIO_ROOT_USER"), SecretKey: os.Getenv("MINIO_ROOT_PASSWORD"), Bucket: bucket})
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger, _ := zap.NewDevelopment()
	admin := NewHandler(db, store, logger)
	public := gallery.NewHandler(db, store, logger)
	router.POST("/admin/gallery", admin.CreateGallery)
	router.PUT("/admin/gallery/:id", admin.UpdateGallery)
	router.POST("/admin/gallery/:id/image", admin.UploadGalleryImage)
	router.DELETE("/admin/gallery/:id", admin.DeleteGallery)
	router.GET("/gallery", public.List)
	router.GET("/gallery/:id/image", public.Image)
	request := func(method, path, contentType string, body []byte) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		result := httptest.NewRecorder()
		router.ServeHTTP(result, req)
		return result
	}
	title := fmt.Sprintf("gallery-test-%d", time.Now().UnixNano())
	create := request(http.MethodPost, "/admin/gallery", "application/json", []byte(fmt.Sprintf(`{"title":%q,"status":"draft"}`, title)))
	var envelope struct {
		Code int          `json:"code"`
		Data gallery.Item `json:"data"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &envelope); err != nil || envelope.Code != 0 || envelope.Data.ID == 0 {
		t.Fatalf("create gallery item: %s error=%v", create.Body.String(), err)
	}
	id := envelope.Data.ID
	defer db.Delete(&gallery.Item{}, id)
	defer store.RemovePrefix(context.Background(), fmt.Sprintf("gallery/%d", id))
	path := fmt.Sprintf("/gallery/%d/image", id)
	if result := request(http.MethodGet, path, "", nil); result.Code != http.StatusNotFound {
		t.Fatalf("draft image status=%d", result.Code)
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 89, G: 142, B: 112, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	var upload bytes.Buffer
	form := multipart.NewWriter(&upload)
	file, err := form.CreateFormFile("file", "test.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write(encoded.Bytes())
	_ = form.Close()
	result := request(http.MethodPost, fmt.Sprintf("/admin/gallery/%d/image", id), form.FormDataContentType(), upload.Bytes())
	if result.Code != http.StatusOK || !strings.Contains(result.Body.String(), `"code":0`) {
		t.Fatalf("upload gallery image: %s", result.Body.String())
	}
	update := request(http.MethodPut, fmt.Sprintf("/admin/gallery/%d", id), "application/json", []byte(fmt.Sprintf(`{"title":%q,"status":"published"}`, title)))
	if !strings.Contains(update.Body.String(), `"code":0`) {
		t.Fatalf("publish gallery image: %s", update.Body.String())
	}
	photo := request(http.MethodGet, path, "", nil)
	if photo.Code != http.StatusOK || photo.Header().Get("Content-Type") != "image/png" || !bytes.Equal(photo.Body.Bytes(), encoded.Bytes()) {
		t.Fatalf("public image status=%d type=%s", photo.Code, photo.Header().Get("Content-Type"))
	}
	list := request(http.MethodGet, "/gallery", "", nil)
	if !strings.Contains(list.Body.String(), title) {
		t.Fatalf("published photo missing: %s", list.Body.String())
	}
	deleted := request(http.MethodDelete, fmt.Sprintf("/admin/gallery/%d", id), "", nil)
	if !strings.Contains(deleted.Body.String(), `"code":0`) {
		t.Fatalf("delete gallery item: %s", deleted.Body.String())
	}
	if result := request(http.MethodGet, path, "", nil); result.Code != http.StatusNotFound {
		t.Fatalf("deleted image status=%d", result.Code)
	}
}
