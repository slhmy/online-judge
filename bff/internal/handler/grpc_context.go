package handler

import (
	"context"
	"net/http"

	"google.golang.org/grpc/metadata"
)

// grpcContextFromRequest forwards authentication header into gRPC metadata.
// Backend is responsible for token validation and claim extraction.
func grpcContextFromRequest(r *http.Request) context.Context {
	pairs := []string{}

	if auth := r.Header.Get("Authorization"); auth != "" {
		pairs = append(pairs, "authorization", auth)
	}

	if len(pairs) == 0 {
		return r.Context()
	}

	md := metadata.Pairs(pairs...)
	return metadata.NewOutgoingContext(r.Context(), md)
}
