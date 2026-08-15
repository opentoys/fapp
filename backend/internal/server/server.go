package server

import (
	"context"
	"net/http"

	"disapp/pkg/token"
	"disapp/internal/resources/config"
	"disapp/internal/resources/storage"

	"gorm.io/gorm"
)

type Server struct {
	DB      *gorm.DB
	Storage storage.Storage
	Config  config.Config
}

func New(gdb *gorm.DB, st storage.Storage, cfg config.Config) *Server {
	return &Server{DB: gdb, Storage: st, Config: cfg}
}

type ctxKey int

const userKey ctxKey = 0

func withUser(ctx context.Context, c *token.Claims) context.Context {
	return context.WithValue(ctx, userKey, c)
}

func userFrom(r *http.Request) *token.Claims {
	c, _ := r.Context().Value(userKey).(*token.Claims)
	return c
}
