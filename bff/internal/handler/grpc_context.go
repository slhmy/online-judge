package handler

import (
	"context"
	"net/http"

	"github.com/slhmy/online-judge/bff/internal/middleware"
	"google.golang.org/grpc/metadata"
)

// grpcContextFromRequest forwards auth/user identity into gRPC metadata.
func grpcContextFromRequest(r *http.Request) context.Context {
	pairs := []string{}

	if auth := r.Header.Get("Authorization"); auth != "" {
		pairs = append(pairs, "authorization", auth)
	}
	if userID := middleware.GetUserID(r.Context()); userID != "" {
		pairs = append(pairs, "x-user-id", userID)
	}
	if userEmail := middleware.GetUserEmail(r.Context()); userEmail != "" {
		pairs = append(pairs, "x-user-email", userEmail)
	}
	if userRole := middleware.GetUserRole(r.Context()); userRole != "" {
		pairs = append(pairs, "x-user-role", userRole)
	}

	if len(pairs) == 0 {
		return r.Context()
	}

	md := metadata.Pairs(pairs...)
	return metadata.NewOutgoingContext(r.Context(), md)
}
