package controller

import (
	"context"
	"net/http"

	"disapp/internal/service"
	"disapp/pkg/token"
)

// Controller holds HTTP handlers. Business logic lives in Service.
type Controller struct {
	SVC *service.Service
}

func New(svc *service.Service) *Controller {
	return &Controller{SVC: svc}
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