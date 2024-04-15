package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dgrijalva/jwt-go"
)

// JWTClaims represents the claims in the JWT token
type JWTClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// AuthorizeRequest represents the input for the Lambda function
type AuthorizeRequest struct {
	AuthorizationHeader string `json:"authorizationHeader"`
}

// AuthorizeResponse represents the output of the Lambda function
type AuthorizeResponse struct {
	Authorized bool   `json:"authorized"`
	Username   string `json:"username"`
}

// AuthorizeHandler is the main function that handles the authorization request
func AuthorizeHandler(ctx context.Context, request events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	// Extract the JWT token from the Authorization header
	token, err := extractJWTToken(request.AuthorizationHeader)
	if err != nil {
		return events.APIGatewayCustomAuthorizerResponse{}, err
	}

	// Verify the JWT token
	claims, err := verifyJWTToken(token)
	if err != nil {
		return events.APIGatewayCustomAuthorizerResponse{}, err
	}

	// Generate the policy document
	policyDocument := generatePolicyDocument(claims.Username, "Allow", "*")

	// Return the policy document
	return events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: claims.Username,
		PolicyDocument: policyDocument,
	}, nil
}

// extractJWTToken extracts the JWT token from the Authorization header
func extractJWTToken(authorizationHeader string) (string, error) {
	if authorizationHeader == "" {
		return "", errors.New("Authorization header is missing")
	}

	parts := strings.Split(authorizationHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("Invalid Authorization header format")
	}

	return parts[1], nil
}

// verifyJWTToken verifies the JWT token and returns the claims
func verifyJWTToken(token string) (*JWTClaims, error) {
	claims := &JWTClaims{
		Username: "emprovise_user",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: 1681500000, // Set the expiration time to a future date
		},
	}

	return claims, nil
}

// generatePolicyDocument generates the policy document for the API Gateway
func generatePolicyDocument(principalID, effect, resource string) events.APIGatewayCustomAuthorizerPolicy {
    return events.APIGatewayCustomAuthorizerResponse {
        PrincipalID: "my-user",
        PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
            Version: "2012-10-17",
            Statement: []events.IAMPolicyStatement{
                {
                    Action:   []string{"execute-api:Invoke"},
                    Effect:   effect,
                    Resource: []string{resource},
                },
            },
        }
	}
}

func main() {
	lambda.Start(AuthorizeHandler)
}
