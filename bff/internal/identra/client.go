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

// GetOAuthAuthorizationURL gets the OAuth authorization URL from Identra
func (c *Client) GetOAuthAuthorizationURL(ctx context.Context, provider string, redirectURL *string) (*pb.GetOAuthAuthorizationURLResponse, error) {
	resp, err := c.client.GetOAuthAuthorizationURL(ctx, &pb.GetOAuthAuthorizationURLRequest{
		Provider:    provider,
		RedirectUrl: redirectURL,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// LoginByOAuth authenticates user via OAuth code and state
func (c *Client) LoginByOAuth(ctx context.Context, code, state string) (*pb.LoginByOAuthResponse, error) {
	resp, err := c.client.LoginByOAuth(ctx, &pb.LoginByOAuthRequest{
		Code:  code,
		State: state,
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
