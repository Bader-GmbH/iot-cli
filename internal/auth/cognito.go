package auth

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// CognitoConfig holds the Cognito configuration
type CognitoConfig struct {
	Region       string
	UserPoolID   string
	ClientID     string
}

// DefaultCognitoConfig returns the default Cognito configuration
func DefaultCognitoConfig() CognitoConfig {
	return CognitoConfig{
		Region:     "eu-central-1",
		UserPoolID: "eu-central-1_SyQyrM9xc",
		ClientID:   "70sngp1h120tni8csqp3s683an",
	}
}

// AuthResult contains the tokens from a successful authentication
type AuthResult struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	ExpiresIn    int32
}

// CognitoClient wraps the Cognito Identity Provider client
type CognitoClient struct {
	client   *cognitoidentityprovider.Client
	config   CognitoConfig
}

// NewCognitoClient creates a new Cognito client
func NewCognitoClient(cfg CognitoConfig) (*CognitoClient, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := cognitoidentityprovider.NewFromConfig(awsCfg)

	return &CognitoClient{
		client: client,
		config: cfg,
	}, nil
}

// Authenticate performs SRP authentication with email and password
func (c *CognitoClient) Authenticate(ctx context.Context, email, password string) (*AuthResult, error) {
	input := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeUserPasswordAuth,
		ClientId: aws.String(c.config.ClientID),
		AuthParameters: map[string]string{
			"USERNAME": email,
			"PASSWORD": password,
		},
	}

	result, err := c.client.InitiateAuth(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	if result.ChallengeName != "" {
		return nil, fmt.Errorf("authentication requires additional challenge: %s", result.ChallengeName)
	}

	if result.AuthenticationResult == nil {
		return nil, fmt.Errorf("authentication failed: no result returned")
	}

	return &AuthResult{
		AccessToken:  aws.ToString(result.AuthenticationResult.AccessToken),
		IDToken:      aws.ToString(result.AuthenticationResult.IdToken),
		RefreshToken: aws.ToString(result.AuthenticationResult.RefreshToken),
		ExpiresIn:    result.AuthenticationResult.ExpiresIn,
	}, nil
}

// RefreshTokens uses the refresh token to get new access and ID tokens
func (c *CognitoClient) RefreshTokens(ctx context.Context, refreshToken string) (*AuthResult, error) {
	input := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeRefreshTokenAuth,
		ClientId: aws.String(c.config.ClientID),
		AuthParameters: map[string]string{
			"REFRESH_TOKEN": refreshToken,
		},
	}

	result, err := c.client.InitiateAuth(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	if result.AuthenticationResult == nil {
		return nil, fmt.Errorf("token refresh failed: no result returned")
	}

	return &AuthResult{
		AccessToken:  aws.ToString(result.AuthenticationResult.AccessToken),
		IDToken:      aws.ToString(result.AuthenticationResult.IdToken),
		RefreshToken: refreshToken, // Cognito doesn't always return a new refresh token
		ExpiresIn:    result.AuthenticationResult.ExpiresIn,
	}, nil
}
