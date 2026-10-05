package store

import "time"

type Product struct {
	ID          uint64    `json:"id" gorm:"primaryKey"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CoverURL    string    `json:"cover_url"`
	PriceCents  int64     `json:"price_cents"`
	Stock       int       `json:"stock"`
	Status      string    `json:"-"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

type Order struct {
	ID         uint64    `json:"-" gorm:"primaryKey"`
	OrderNo    string    `json:"order_no"`
	ProductID  uint64    `json:"product_id"`
	Email      string    `json:"email"`
	Quantity   int       `json:"quantity"`
	TotalCents int64     `json:"total_cents"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"-"`
}

type CreateOrderRequest struct {
	ProductID uint64 `json:"product_id" binding:"required"`
	Email     string `json:"email" binding:"required,email,max=255"`
	Quantity  int    `json:"quantity" binding:"required,min=1,max=10"`
}
