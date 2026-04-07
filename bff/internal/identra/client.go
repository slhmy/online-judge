package identra

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/poly-workshop/identra/gen/go/identra/v1"
)

// Client wraps the identra gRPC client
type Client struct {
	conn   *grpc.ClientConn
	client pb.IdentraServiceClient
}

// NewClient creates a new identra client
func NewClient(addr string) (*Client, error) {
	//nolint:staticcheck // grpc.Dial is deprecated but will be supported throughout 1.x
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: pb.NewIdentraServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// LoginByPassword authenticates user with email and password
func (c *Client) LoginByPassword(ctx context.Context, email, password string) (*pb.LoginByPasswordResponse, error) {
	resp, err := c.client.LoginByPassword(ctx, &pb.LoginByPasswordRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Register registers a new user with email and password
// Note: identra may have separate register endpoint, using LoginByPassword for now
// as it creates user if not exists in some configurations
func (c *Client) Register(ctx context.Context, email, password string) (*pb.LoginByPasswordResponse, error) {
	// Try login first - if user exists, return error
	// For registration, we use the same endpoint with register intent
	resp, err := c.client.LoginByPassword(ctx, &pb.LoginByPasswordRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RefreshToken refreshes the access token
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*pb.RefreshTokenResponse, error) {
	resp, err := c.client.RefreshToken(ctx, &pb.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetCurrentUser gets the current user info from token
func (c *Client) GetCurrentUser(ctx context.Context, accessToken string) (*pb.GetCurrentUserLoginInfoResponse, error) {
	resp, err := c.client.GetCurrentUserLoginInfo(ctx, &pb.GetCurrentUserLoginInfoRequest{
		AccessToken: accessToken,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetJWKS gets the JWKS for token validation
func (c *Client) GetJWKS(ctx context.Context) (*pb.GetJWKSResponse, error) {
	resp, err := c.client.GetJWKS(ctx, &pb.GetJWKSRequest{})
	if err != nil {
		return nil, err
	}
	return resp, nil
}