// Command mcp is the Recurso MCP (Model Context Protocol) server. It exposes a
// curated, tier-gated set of billing tools over Streamable HTTP; each request
// authenticates with the tenant's own rsk_ API key, which is forwarded to the
// /v1 API. It holds no database connection of its own.
//
// Config (env):
//
//	API_BASE_URL  base URL of the /v1 API (default http://localhost:8080)
//	PORT          port to listen on          (default 8090)
//	MCP_VERSION   server version string       (default 0.1.0)
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
	addr := ":" + getenv("PORT", "8090")

	client := appmcp.NewClient(base)
	srv := appmcp.NewServer(client, appmcp.Options{Version: getenv("MCP_VERSION", "0.1.0")})

	// One shared MCP server serves every request; per-caller auth flows through
	// the request headers into each tool handler.
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
		log.Printf("recurso-mcp listening on %s → /v1 at %s", addr, base)
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
