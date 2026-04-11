package middleware

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/jwk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	contextKeyUserID contextKey = "user_id"
	contextKeyEmail  contextKey = "user_email"
	contextKeyRole   contextKey = "user_role"
)

type JWTInterceptor struct {
	jwksURL string
	keySet  jwk.Set
}

func NewJWTInterceptor(jwksURL string) (*JWTInterceptor, error) {
	// Fetch JWKS
	set, err := jwk.Fetch(context.Background(), jwksURL)
	if err != nil {
		return nil, err
	}

	return &JWTInterceptor{
		jwksURL: jwksURL,
		keySet:  set,
	}, nil
}

// Unary returns a unary server interceptor for JWT validation
func (i *JWTInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract token from metadata when present.
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}

		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return handler(ctx, req)
		}

		token := extractBearerToken(authHeader[0])
		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		// Validate token
		claims, err := i.validateToken(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		// Add user info to context
		ctx = context.WithValue(ctx, contextKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, contextKeyEmail, claims.Email)
		ctx = context.WithValue(ctx, contextKeyRole, claims.Role)

		return handler(ctx, req)
	}
}

type UserClaims struct {
	UserID string
	Email  string
	Role   string
}

func (i *JWTInterceptor) validateToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}

		// Get key ID from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, jwt.ErrSignatureInvalid
		}

		// Find matching key in JWKS
		key, ok := i.keySet.LookupKeyID(kid)
		if !ok {
			return nil, jwt.ErrSignatureInvalid
		}

		// Extract public key
		var pubkey interface{}
		if err := key.Raw(&pubkey); err != nil {
			return nil, err
		}

		return pubkey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrSignatureInvalid
	}

	userID, _ := claims["user_id"].(string)
	if userID == "" {
		if sub, ok := claims["sub"].(string); ok {
			userID = sub
		}
	}

	email, _ := claims["email"].(string)
	role, _ := claims["role"].(string)
	if role == "" {
		if roles, ok := claims["roles"].([]interface{}); ok {
			for _, v := range roles {
				if rs, ok := v.(string); ok && rs != "" {
					role = rs
					if rs == "admin" {
						break
					}
				}
			}
		}
	}

	return &UserClaims{UserID: userID, Email: email, Role: role}, nil
}

func extractBearerToken(header string) string {
	if len(header) < 7 || !strings.EqualFold(header[:7], "Bearer ") {
		return ""
	}
	return header[7:]
}

// ContextWithUserID returns a new context with the user ID set.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, contextKeyUserID, userID)
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(contextKeyUserID).(string); ok {
		return userID
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for _, key := range []string{"x-user-id", "user_id", "user-id"} {
			if vals := md.Get(key); len(vals) > 0 && vals[0] != "" {
				return vals[0]
			}
		}
	}
	return ""
}

// GetUserEmail extracts user email from context
func GetUserEmail(ctx context.Context) string {
	if email, ok := ctx.Value(contextKeyEmail).(string); ok {
		return email
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for _, key := range []string{"x-user-email", "user_email", "user-email"} {
			if vals := md.Get(key); len(vals) > 0 && vals[0] != "" {
				return vals[0]
			}
		}
	}
	return ""
}

// GetUserRole extracts user role from context.
func GetUserRole(ctx context.Context) string {
	if role, ok := ctx.Value(contextKeyRole).(string); ok {
		return role
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for _, key := range []string{"x-user-role", "user_role", "user-role"} {
			if vals := md.Get(key); len(vals) > 0 && vals[0] != "" {
				return vals[0]
			}
		}
	}
	return ""
}
