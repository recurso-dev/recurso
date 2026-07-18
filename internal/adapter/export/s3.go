// Package export ships finance data to operator-owned object storage
// (Track D5, spec_lago_parity.md). The destination is the operator's own
// bucket — operator-configured egress in the same class as SMTP and
// webhooks — so it is not residency-blocked; leaving the config unset
// disables it entirely.
package export

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// S3Client is a minimal, dependency-free S3 PutObject client using AWS
// Signature V4. Endpoint is overridable for S3-compatible stores (MinIO,
// Cloudflare R2) and tests.
type S3Client struct {
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	// Endpoint overrides https://<bucket>.s3.<region>.amazonaws.com with a
	// path-style endpoint (http://minio:9000). Empty = AWS virtual-hosted.
	Endpoint   string
	HTTPClient *http.Client
	now        func() time.Time
}

// NewS3Client builds a client; any empty required field disables callers.
func NewS3Client(bucket, region, accessKey, secretKey, endpoint string) *S3Client {
	return &S3Client{
		Bucket:     bucket,
		Region:     region,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		Endpoint:   endpoint,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
		now:        func() time.Time { return time.Now().UTC() },
	}
}

// Configured reports whether the client can actually upload.
func (c *S3Client) Configured() bool {
	return c != nil && c.Bucket != "" && c.Region != "" && c.AccessKey != "" && c.SecretKey != ""
}

// PutObject uploads body to key with the given content type.
func (c *S3Client) PutObject(ctx context.Context, key string, body []byte, contentType string) error {
	if !c.Configured() {
		return fmt.Errorf("s3 export not configured")
	}
	key = strings.TrimPrefix(key, "/")

	var host, path string
	if c.Endpoint != "" {
		host = strings.TrimPrefix(strings.TrimPrefix(c.Endpoint, "https://"), "http://")
		path = "/" + c.Bucket + "/" + key
	} else {
		host = fmt.Sprintf("%s.s3.%s.amazonaws.com", c.Bucket, c.Region)
		path = "/" + key
	}
	scheme := "https"
	if strings.HasPrefix(c.Endpoint, "http://") {
		scheme = "http"
	}

	now := c.now()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	payloadHash := sha256Hex(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, scheme+"://"+host+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Host", host)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	// --- SigV4 ---
	canonicalHeaders := "content-type:" + contentType + "\n" +
		"host:" + host + "\n" +
		"x-amz-content-sha256:" + payloadHash + "\n" +
		"x-amz-date:" + amzDate + "\n"
	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"
	canonicalRequest := strings.Join([]string{
		http.MethodPut, uriEncodePath(path), "", canonicalHeaders, signedHeaders, payloadHash,
	}, "\n")

	scope := strings.Join([]string{dateStamp, c.Region, "s3", "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256", amzDate, scope, sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := hmacSHA256(hmacSHA256(hmacSHA256(hmacSHA256(
		[]byte("AWS4"+c.SecretKey), dateStamp), c.Region), "s3"), "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.AccessKey, scope, signedHeaders, signature,
	))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("s3 put %s: %w", key, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("s3 put %s: HTTP %d: %s", key, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

// uriEncodePath applies AWS's URI encoding to each path segment (segment
// separators preserved; unreserved characters untouched).
func uriEncodePath(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		segments[i] = uriEncode(seg)
	}
	return strings.Join(segments, "/")
}

func uriEncode(s string) string {
	var b strings.Builder
	for _, ch := range []byte(s) {
		switch {
		case ch >= 'A' && ch <= 'Z', ch >= 'a' && ch <= 'z', ch >= '0' && ch <= '9',
			ch == '-', ch == '_', ch == '.', ch == '~':
			b.WriteByte(ch)
		default:
			fmt.Fprintf(&b, "%%%02X", ch)
		}
	}
	return b.String()
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
