package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("product not found")
var ErrOutOfStock = errors.New("product out of stock")
var ErrOrderNotFound = errors.New("order not found")
var ErrOrderNotCancellable = errors.New("order cannot be cancelled")

type OrderSummary struct {
	Order
	ProductName string `json:"product_name"`
}

type Repository interface {
	ListProducts(context.Context, int, int) ([]Product, int64, error)
	GetProduct(context.Context, string) (*Product, error)
	CreateOrder(context.Context, CreateOrderRequest) (*Order, error)
	ListOrdersForUser(context.Context, uint64, int, int) ([]OrderSummary, int64, error)
	CancelOrder(context.Context, uint64, uint64) (*Order, error)
}

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &repository{db} }

func (r *repository) ListProducts(ctx context.Context, limit, offset int) ([]Product, int64, error) {
	q := r.db.WithContext(ctx).Model(&Product{}).Where("status = ?", "published")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}
	products := make([]Product, 0)
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&products).Error; err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	return products, total, nil
}

func (r *repository) GetProduct(ctx context.Context, slug string) (*Product, error) {
	var product Product
	err := r.db.WithContext(ctx).Where("slug = ? AND status = ?", slug, "published").Take(&product).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	return &product, nil
}

func (r *repository) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
	if req.UserID == 0 || strings.TrimSpace(req.Email) == "" {
		return nil, errors.New("order requires an authenticated user")
	}
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("generate order number: %w", err)
	}
	orderNo := strings.ToUpper(hex.EncodeToString(bytes))
	var order Order
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var product Product
		if err := tx.Where("id = ? AND status = ?", req.ProductID, "published").Take(&product).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("get order product: %w", err)
		}
		if product.PriceCents > 0 && int64(req.Quantity) > (1<<63-1)/product.PriceCents {
			return fmt.Errorf("order total overflow")
		}
		updated := tx.Model(&Product{}).Where("id = ? AND status = ? AND stock >= ?", req.ProductID, "published", req.Quantity).Update("stock", gorm.Expr("stock - ?", req.Quantity))
		if updated.Error != nil {
			return fmt.Errorf("reserve stock: %w", updated.Error)
		}
		if updated.RowsAffected == 0 {
			return ErrOutOfStock
		}
		order = Order{OrderNo: orderNo, UserID: &req.UserID, ProductID: req.ProductID, Email: strings.ToLower(strings.TrimSpace(req.Email)), Quantity: req.Quantity, TotalCents: product.PriceCents * int64(req.Quantity), Status: "pending_payment"}
		if err := tx.Create(&order).Error; err != nil {
			return fmt.Errorf("create order: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *repository) ListOrdersForUser(ctx context.Context, userID uint64, limit, offset int) ([]OrderSummary, int64, error) {
	q := r.db.WithContext(ctx).Model(&Order{}).Where("orders.user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count user orders: %w", err)
	}
	items := make([]OrderSummary, 0)
	if err := q.Select("orders.*, products.name AS product_name").Joins("JOIN products ON products.id = orders.product_id").Order("orders.created_at DESC, orders.id DESC").Limit(limit).Offset(offset).Scan(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list user orders: %w", err)
	}
	return items, total, nil
}

func (r *repository) CancelOrder(ctx context.Context, userID, orderID uint64) (*Order, error) {
	var order Order
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", orderID, userID).Take(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return fmt.Errorf("find user order: %w", err)
		}
		if order.Status != "pending_payment" {
			return ErrOrderNotCancellable
		}
		if err := tx.Model(&Product{}).Where("id = ?", order.ProductID).Update("stock", gorm.Expr("stock + ?", order.Quantity)).Error; err != nil {
			return fmt.Errorf("release order stock: %w", err)
		}
		order.Status = "cancelled"
		return tx.Save(&order).Error
	})
	if err != nil {
		return nil, err
	}
	return &order, nil
}
