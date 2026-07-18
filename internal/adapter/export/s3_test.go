package export

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testClient(srvURL string) *S3Client {
	c := NewS3Client("finance", "ap-south-1", "AKIDEXAMPLE", "secret", srvURL)
	c.now = func() time.Time { return time.Date(2026, 7, 19, 3, 0, 0, 0, time.UTC) }
	return c
}

func TestPutObjectSignsAndUploads(t *testing.T) {
	var got struct {
		path, auth, amzDate, contentSHA, contentType string
		body                                         []byte
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		got.auth = r.Header.Get("Authorization")
		got.amzDate = r.Header.Get("X-Amz-Date")
		got.contentSHA = r.Header.Get("X-Amz-Content-Sha256")
		got.contentType = r.Header.Get("Content-Type")
		got.body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	body := []byte("transaction_id,amount\ntx1,100\n")
	if err := c.PutObject(context.Background(), "t1/general-ledger-2026-07-19.csv", body, "text/csv"); err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	// Path-style addressing against the custom endpoint.
	if got.path != "/finance/t1/general-ledger-2026-07-19.csv" {
		t.Fatalf("path = %q", got.path)
	}
	if string(got.body) != string(body) || got.contentType != "text/csv" {
		t.Fatalf("body/type not forwarded")
	}
	// Payload hash must be the real SHA-256, not UNSIGNED-PAYLOAD.
	sum := sha256.Sum256(body)
	if got.contentSHA != hex.EncodeToString(sum[:]) {
		t.Fatalf("content sha = %q", got.contentSHA)
	}
	// SigV4 authorization: algorithm, credential scope, signed headers, signature.
	if !strings.HasPrefix(got.auth, "AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE/20260719/ap-south-1/s3/aws4_request, ") {
		t.Fatalf("auth = %q", got.auth)
	}
	if !strings.Contains(got.auth, "SignedHeaders=content-type;host;x-amz-content-sha256;x-amz-date") {
		t.Fatalf("signed headers missing: %q", got.auth)
	}
	if !strings.Contains(got.auth, "Signature=") || got.amzDate != "20260719T030000Z" {
		t.Fatalf("signature/date missing: %q %q", got.auth, got.amzDate)
	}
}

func TestPutObjectSignatureDeterministic(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := testClient(srv.URL)

	sig := func() string {
		if err := c.PutObject(context.Background(), "k.csv", []byte("x"), "text/csv"); err != nil {
			t.Fatalf("PutObject: %v", err)
		}
		return auth[strings.Index(auth, "Signature="):]
	}
	first := sig()
	second := sig()
	if first != second {
		t.Fatal("same inputs against the same host must produce the same signature")
	}
}

func TestPutObjectSurfacesHTTPErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<Error><Code>SignatureDoesNotMatch</Code></Error>`))
	}))
	defer srv.Close()
	c := testClient(srv.URL)
	err := c.PutObject(context.Background(), "k", []byte("x"), "text/csv")
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("err = %v, want HTTP 403 surfaced", err)
	}
}

func TestUnconfiguredClientRefuses(t *testing.T) {
	c := NewS3Client("", "", "", "", "")
	if c.Configured() {
		t.Fatal("empty config must not report configured")
	}
	if err := c.PutObject(context.Background(), "k", nil, "text/csv"); err == nil {
		t.Fatal("unconfigured client must refuse uploads")
	}
}
