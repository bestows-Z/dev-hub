package media

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	"github.com/bestows-Z/dev-hub/backend/internal/storage"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCoverUploadServeDeleteIntegration(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_MEDIA") != "1" {
		t.Skip("set DEVHUB_TEST_MEDIA=1 to test PostgreSQL and MinIO")
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
	suffix := time.Now().UnixNano()
	u := user.User{Username: fmt.Sprintf("media_%d", suffix), Email: fmt.Sprintf("media_%d@example.invalid", suffix), PasswordHash: "test-only", Role: user.RoleAdmin, Status: user.StatusNormal}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&u)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("currentUser", &u); c.Next() })
	handler := NewHandler(db, store, zap.NewNop())
	router.POST("/admin/media", handler.Upload)
	router.DELETE("/admin/media/:id", handler.Delete)
	router.GET("/media/:id", handler.Serve)
	imageBody := image.NewRGBA(image.Rect(0, 0, 2, 2))
	imageBody.Set(0, 0, color.RGBA{R: 22, G: 82, B: 55, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, imageBody); err != nil {
		t.Fatal(err)
	}
	var payload bytes.Buffer
	form := multipart.NewWriter(&payload)
	file, err := form.CreateFormFile("file", "cover.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(file, bytes.NewReader(encoded.Bytes()))
	_ = form.Close()
	req := httptest.NewRequest(http.MethodPost, "/admin/media", &payload)
	req.Header.Set("Content-Type", form.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var result struct {
		Code int                      `json:"code"`
		Data struct{ ID, URL string } `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Code != 0 || result.Data.ID == "" {
		t.Fatalf("upload: %s err=%v", w.Body.String(), err)
	}
	defer func() {
		var asset Asset
		if db.First(&asset, "id = ?", result.Data.ID).Error == nil {
			_ = store.Remove(t.Context(), asset.ObjectKey)
			_ = db.Delete(&asset).Error
		}
	}()
	if result.Data.URL != "/api/v1/media/"+result.Data.ID {
		t.Fatalf("unexpected URL: %q", result.Data.URL)
	}
	imageResponse := httptest.NewRecorder()
	router.ServeHTTP(imageResponse, httptest.NewRequest(http.MethodGet, "/media/"+result.Data.ID, nil))
	if imageResponse.Code != http.StatusOK || imageResponse.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("serve: %d %s", imageResponse.Code, imageResponse.Body.String())
	}
	if _, _, err := image.Decode(bytes.NewReader(imageResponse.Body.Bytes())); err != nil {
		t.Fatalf("image cannot decode: %v", err)
	}
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, httptest.NewRequest(http.MethodDelete, "/admin/media/"+result.Data.ID, nil))
	if deleteResponse.Code != http.StatusOK || !bytes.Contains(deleteResponse.Body.Bytes(), []byte(`"deleted":true`)) {
		t.Fatalf("delete: %s", deleteResponse.Body.String())
	}
}
