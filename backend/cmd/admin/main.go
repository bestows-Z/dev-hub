package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	pg "github.com/bestows-Z/dev-hub/backend/internal/platform/postgres"
	"github.com/bestows-Z/dev-hub/backend/internal/user"
	"go.uber.org/zap"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "promote" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/admin promote <username>")
		os.Exit(2)
	}
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger := zap.NewNop()
	db, err := pg.New(pg.Config{DSN: cfg.Postgres.DSN(), MaxOpenConns: 2, MaxIdleConns: 1}, logger)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	name := strings.ToLower(strings.TrimSpace(os.Args[2]))
	result := db.DB.Model(&user.User{}).Where("username = ? AND status = ?", name, user.StatusNormal).Update("role", user.RoleAdmin)
	if result.Error != nil {
		panic(result.Error)
	}
	if result.RowsAffected != 1 {
		fmt.Fprintln(os.Stderr, "active user not found")
		os.Exit(1)
	}
	fmt.Printf("%s is now an administrator\n", name)
}
