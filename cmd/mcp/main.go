// Command mcp is the Recurso MCP (Model Context Protocol) server. It exposes a
// curated, tier-gated set of billing tools; each request authenticates with a
// tenant rsk_ API key that is forwarded to the /v1 API. It holds no database
// connection of its own.
//
// Two transports:
//
//	http  (default) — remote Streamable HTTP, multi-tenant. Each caller sends
//	                   their own key as `Authorization: Bearer rsk_...`.
//	stdio           — local single-tenant (e.g. Claude Desktop). The key comes
//	                   from RECURSO_API_KEY and is used for every call.
//
// Config (env):
//
//	MCP_TRANSPORT   http | stdio               (default http)
//	API_BASE_URL    base URL of the /v1 API     (default http://localhost:8080)
//	PORT            http listen port            (default 8090)
//	RECURSO_API_KEY rsk_ key for stdio mode     (required when MCP_TRANSPORT=stdio)
//	MCP_VERSION     server version string       (default 0.1.0)
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	appmcp "github.com/recurso-dev/recurso/internal/mcp"
)

func main() {
	base := getenv("API_BASE_URL", "http://localhost:8080")
	client := appmcp.NewClient(base)

	switch getenv("MCP_TRANSPORT", "http") {
	case "stdio":
		runStdio(client, base)
	default:
		runHTTP(client, base)
	}
}

// runStdio serves a single tenant over stdin/stdout, using RECURSO_API_KEY for
// every call. Intended for local clients like Claude Desktop.
func runStdio(client *appmcp.Client, base string) {
	key := os.Getenv("RECURSO_API_KEY")
	if key == "" {
		log.Fatal("stdio transport requires RECURSO_API_KEY")
	}
	srv := appmcp.NewServer(client, appmcp.Options{
		Version:   getenv("MCP_VERSION", "0.1.0"),
		StaticKey: key,
	})
	log.Printf("recurso-mcp (stdio) → /v1 at %s", base)
	if err := srv.MCP().Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("mcp stdio: %v", err)
	}
}

// runHTTP serves many tenants over Streamable HTTP; per-caller auth flows
// through the request headers into each tool handler.
func runHTTP(client *appmcp.Client, base string) {
	addr := ":" + getenv("PORT", "8090")
	srv := appmcp.NewServer(client, appmcp.Options{Version: getenv("MCP_VERSION", "0.1.0")})

	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv.MCP()
	}, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/", mcpHandler)

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("recurso-mcp (http) listening on %s → /v1 at %s", addr, base)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("mcp server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	log.Println("recurso-mcp stopped")
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
