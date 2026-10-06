package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestOrderOwnershipAndCancellationIntegration(t *testing.T) {
	if os.Getenv("DEVHUB_TEST_ORDERS") != "1" {
		t.Skip("set DEVHUB_TEST_ORDERS=1 to test PostgreSQL orders")
	}
	cfg := config.PostgresConfig{Host: "127.0.0.1", Port: 5432, User: os.Getenv("POSTGRES_USER"), Password: os.Getenv("POSTGRES_PASSWORD"), DBName: os.Getenv("POSTGRES_DB"), SSLMode: "disable", TimeZone: "UTC"}
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	suffix := time.Now().UnixNano()
	u := user.User{Username: fmt.Sprintf("order_test_%d", suffix), Email: fmt.Sprintf("order_test_%d@example.invalid", suffix), PasswordHash: "integration-test-only", Role: user.RoleUser, Status: user.StatusNormal}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&u)
	p := Product{Slug: fmt.Sprintf("order-test-%d", suffix), Name: "Order integration item", PriceCents: 1299, Stock: 2, Status: "published"}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&p)
	repo := NewRepository(db)
	order, err := repo.CreateOrder(context.Background(), CreateOrderRequest{ProductID: p.ID, Quantity: 1, UserID: u.ID, Email: u.Email})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Delete(order)
	if order.UserID == nil || *order.UserID != u.ID || order.TotalCents != 1299 {
		t.Fatalf("wrong ownership or total: %+v", order)
	}
	items, total, err := repo.ListOrdersForUser(context.Background(), u.ID, 10, 0)
	if err != nil || total != 1 || len(items) != 1 || items[0].ProductName != p.Name {
		t.Fatalf("own orders: total=%d items=%+v err=%v", total, items, err)
	}
	if _, err := repo.CancelOrder(context.Background(), u.ID+1, order.ID); !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("other user cancelled order: %v", err)
	}
	if _, err := repo.CancelOrder(context.Background(), u.ID, order.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CancelOrder(context.Background(), u.ID, order.ID); !errors.Is(err, ErrOrderNotCancellable) {
		t.Fatalf("second cancellation: %v", err)
	}
	if err := db.First(&p, p.ID).Error; err != nil || p.Stock != 2 {
		t.Fatalf("stock after cancellation=%d err=%v", p.Stock, err)
	}
}
