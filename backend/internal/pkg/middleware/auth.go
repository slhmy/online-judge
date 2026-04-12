package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lestrrat-go/jwx/jwk"
	identra_v1_pb "github.com/poly-workshop/identra/gen/go/identra/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
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
	db      *pgxpool.Pool
}

func NewJWTInterceptor(jwksURL string, db *pgxpool.Pool) (*JWTInterceptor, error) {
	// Fetch JWKS
	set, err := jwk.Fetch(context.Background(), jwksURL)
	if err != nil {
		// Fallback for deployments that only expose Identra gRPC.
		set, err = fetchJWKSViaIdentraGRPC(jwksURL)
		if err != nil {
			return nil, fmt.Errorf("fetch jwks: %w", err)
		}
	}

	return &JWTInterceptor{
		jwksURL: jwksURL,
		keySet:  set,
		db:      db,
	}, nil
}

func (i *JWTInterceptor) lookupUserRole(ctx context.Context, userID string) (string, error) {
	if i.db == nil || strings.TrimSpace(userID) == "" {
		return "", nil
	}

	var role string
	err := i.db.QueryRow(ctx, `SELECT COALESCE(role, '') FROM user_profiles WHERE user_id = $1`, userID).Scan(&role)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(role), nil
}

func fetchJWKSViaIdentraGRPC(jwksURL string) (jwk.Set, error) {
	u, err := url.Parse(jwksURL)
	if err != nil {
		return nil, err
	}

	addr := strings.TrimSpace(u.Host)
	if addr == "" {
		addr = strings.TrimSpace(jwksURL)
	}
	if addr == "" {
		return nil, fmt.Errorf("empty identra grpc address")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//nolint:staticcheck // grpc.Dial is deprecated but supported in grpc 1.x
	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("dial identra grpc %q: %w", addr, err)
	}
	defer func() { _ = conn.Close() }()

	resp, err := identra_v1_pb.NewIdentraServiceClient(conn).GetJWKS(ctx, &identra_v1_pb.GetJWKSRequest{})
	if err != nil {
		return nil, fmt.Errorf("identra GetJWKS: %w", err)
	}

	type jwkKey struct {
		Kty string `json:"kty"`
		Alg string `json:"alg,omitempty"`
		Use string `json:"use,omitempty"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
	}
	type jwksDoc struct {
		Keys []jwkKey `json:"keys"`
	}

	doc := jwksDoc{Keys: make([]jwkKey, 0, len(resp.GetKeys()))}
	for _, key := range resp.GetKeys() {
		doc.Keys = append(doc.Keys, jwkKey{
			Kty: strings.TrimSpace(key.GetKty()),
			Alg: strings.TrimSpace(key.GetAlg()),
			Use: strings.TrimSpace(key.GetUse()),
			Kid: strings.TrimSpace(key.GetKid()),
			N:   strings.TrimSpace(key.GetN()),
			E:   strings.TrimSpace(key.GetE()),
		})
	}

	buf, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}

	set, err := jwk.Parse(buf)
	if err != nil {
		return nil, err
	}
	return set, nil
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

		if strings.TrimSpace(claims.Role) == "" {
			if role, err := i.lookupUserRole(ctx, claims.UserID); err == nil && role != "" {
				claims.Role = role
			}
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
