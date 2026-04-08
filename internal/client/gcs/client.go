package gcs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/bina-marga/survey-photo/internal/service"
)

// Client implements the service.GCSClient interface
type Client struct {
	client         *storage.Client
	bucket         *storage.BucketHandle
	bucketName     string
	serviceAccount *ServiceAccountInfo
	testPrefix     string
}

// ServiceAccountInfo holds parsed service account details
type ServiceAccountInfo struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

// NewClient creates a new GCS client
func NewClient(ctx context.Context, config Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Read and parse service account JSON
	saData, err := os.ReadFile(config.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read credentials file: %v", ErrInvalidCredentials, err)
	}

	var serviceAccount ServiceAccountInfo
	if err := json.Unmarshal(saData, &serviceAccount); err != nil {
		return nil, fmt.Errorf("%w: failed to parse credentials file: %v", ErrInvalidCredentials, err)
	}

	if serviceAccount.ClientEmail == "" || serviceAccount.PrivateKey == "" {
		return nil, fmt.Errorf("%w: missing client_email or private_key in credentials", ErrInvalidCredentials)
	}

	// Create storage client with credentials
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(config.CredentialsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	bucket := client.Bucket(config.BucketName)

	return &Client{
		client:         client,
		bucket:         bucket,
		bucketName:     config.BucketName,
		serviceAccount: &serviceAccount,
		testPrefix:     config.TestPrefix,
	}, nil
}

// GenerateSignedURL generates a signed URL for uploading or downloading
func (c *Client) GenerateSignedURL(objectName string, contentType string, expiryMinutes int) (string, error) {
	if objectName == "" {
		return "", fmt.Errorf("object name cannot be empty")
	}

	fullObjectName := c.testPrefix + objectName

	if expiryMinutes <= 0 {
		expiryMinutes = 15
	}

	expiry := time.Now().Add(time.Duration(expiryMinutes) * time.Minute)

	method := http.MethodPut
	if contentType == "" {
		method = http.MethodGet
	}

	opts := &storage.SignedURLOptions{
		GoogleAccessID: c.serviceAccount.ClientEmail,
		PrivateKey:     []byte(c.serviceAccount.PrivateKey),
		Method:         method,
		Expires:        expiry,
		ContentType:    contentType,
	}

	url, err := storage.SignedURL(c.bucketName, fullObjectName, opts)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSignedURLFailed, err)
	}

	return url, nil
}

// FileExists checks if a file exists in GCS
func (c *Client) FileExists(objectName string) (bool, error) {
	if objectName == "" {
		return false, fmt.Errorf("object name cannot be empty")
	}

	fullObjectName := c.testPrefix + objectName

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	obj := c.bucket.Object(fullObjectName)
	_, err := obj.Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// DeleteFile deletes a file from GCS
func (c *Client) DeleteFile(objectName string) error {
	if objectName == "" {
		return fmt.Errorf("object name cannot be empty")
	}

	fullObjectName := c.testPrefix + objectName

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	obj := c.bucket.Object(fullObjectName)
	if err := obj.Delete(ctx); err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil
		}
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// GenerateDownloadURL generates a signed URL for downloading a file (GET method).
// Returns ErrObjectNotFound if the object does not exist in GCS.
func (c *Client) GenerateDownloadURL(objectName string, expiryMinutes int) (string, error) {
	if objectName == "" {
		return "", fmt.Errorf("object name cannot be empty")
	}

	fullObjectName := c.testPrefix + objectName

	if expiryMinutes <= 0 {
		expiryMinutes = 15
	}

	exists, err := c.FileExists(objectName)
	if err != nil {
		return "", fmt.Errorf("failed to check object existence: %w", err)
	}
	if !exists {
		return "", ErrObjectNotFound
	}

	expiry := time.Now().Add(time.Duration(expiryMinutes) * time.Minute)

	opts := &storage.SignedURLOptions{
		GoogleAccessID: c.serviceAccount.ClientEmail,
		PrivateKey:     []byte(c.serviceAccount.PrivateKey),
		Method:         http.MethodGet,
		Expires:        expiry,
	}

	url, err := storage.SignedURL(c.bucketName, fullObjectName, opts)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSignedURLFailed, err)
	}

	return url, nil
}

// GetPublicURL returns the public URL for a GCS object
func (c *Client) GetPublicURL(objectName string) string {
	fullObjectName := c.testPrefix + objectName
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", c.bucketName, fullObjectName)
}

// Close closes the GCS client connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Compile-time interface check - ensures *Client implements service.GCSClient
var _ service.GCSClient = (*Client)(nil)
