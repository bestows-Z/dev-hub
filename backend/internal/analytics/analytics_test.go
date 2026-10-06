package analytics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

func TestTrackerRecordsOnlySuccessfulContentViews(t *testing.T) {
	gin.SetMode(gin.TestMode)
	broker := &Broker{events: make(chan ViewEvent, 4), logger: zap.NewNop()}
	router := gin.New()
	router.Use(Tracker(broker))
	router.GET("/api/v1/articles/:slug", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/v1/products/:slug", func(c *gin.Context) { c.Status(http.StatusNotFound) })
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/articles/example", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/products/missing", nil))
	if len(broker.events) != 1 {
		t.Fatalf("expected one successful view, got %d", len(broker.events))
	}
	view := <-broker.events
	if view.Kind != "article" || view.Slug != "example" {
		t.Fatalf("unexpected view: %+v", view)
	}
}

func TestRabbitMQToMongoDBIntegration(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_ANALYTICS") != "1" {
		t.Skip("set DEVHUB_TEST_ANALYTICS=1 to test RabbitMQ and MongoDB")
	}
	cfg := config.AnalyticsConfig{
		MongoAddr: os.Getenv("MONGO_ADDR"), MongoUser: os.Getenv("MONGO_ROOT_USERNAME"), MongoPassword: os.Getenv("MONGO_ROOT_PASSWORD"),
		RabbitAddr: os.Getenv("RABBITMQ_ADDR"), RabbitUser: os.Getenv("RABBITMQ_USERNAME"), RabbitPassword: os.Getenv("RABBITMQ_PASSWORD"),
	}
	if cfg.MongoAddr == "" {
		cfg.MongoAddr = "127.0.0.1:27017"
	}
	if cfg.RabbitAddr == "" {
		cfg.RabbitAddr = "127.0.0.1:5672"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	store, err := NewStore(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(context.Background())
	broker, err := NewBroker(cfg, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()
	go broker.RunPublisher(ctx)
	go broker.RunConsumer(ctx, store)
	event := ViewEvent{ID: uuid.NewString(), Kind: "article", Slug: "analytics-integration-test", At: time.Now().UTC()}
	defer store.collection.DeleteOne(context.Background(), bson.D{{Key: "_id", Value: event.ID}})
	broker.events <- event
	for ctx.Err() == nil {
		var actual ViewEvent
		err := store.collection.FindOne(ctx, bson.D{{Key: "_id", Value: event.ID}}).Decode(&actual)
		if err == nil {
			if actual.Kind != event.Kind || actual.Slug != event.Slug {
				t.Fatalf("stored wrong event: %+v", actual)
			}
			summary, err := store.Summary(ctx)
			if err != nil || summary.Today < 1 || summary.Last7Days < 1 {
				t.Fatalf("analytics summary: %+v error=%v", summary, err)
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("event was not delivered from RabbitMQ to MongoDB")
}
