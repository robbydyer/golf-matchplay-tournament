package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"scoring-backend/internal/auth"
	"scoring-backend/internal/email"
	"scoring-backend/internal/handlers"
	"scoring-backend/internal/middleware"
	"scoring-backend/internal/store"
	"strings"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	allowedOrigin := os.Getenv("CORS_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5173"
	}

	// Choose store backend via STORE_BACKEND env var.
	storeBackend := os.Getenv("STORE_BACKEND")
	var s store.Store
	switch storeBackend {
	case "file":
		dataDir := os.Getenv("DATA_DIR")
		if dataDir == "" {
			dataDir = "./data"
		}
		fs, err := store.NewFileStore(dataDir)
		if err != nil {
			log.Fatalf("Failed to initialize file store: %v", err)
		}
		s = fs
		log.Printf("Using file store (dir: %s)", dataDir)
	case "firestore":
		log.Fatal("Firestore backend not yet implemented. See internal/store/firestore.go for guidance.")
	default:
		s = store.NewMemoryStore()
		log.Println("Using in-memory store")
	}

	devMode := os.Getenv("DEV_MODE") == "true"

	// Parse admin emails from comma-separated env var
	adminEmails := make(map[string]bool)
	if raw := os.Getenv("ADMIN_EMAILS"); raw != "" {
		for _, em := range strings.Split(raw, ",") {
			em = strings.TrimSpace(strings.ToLower(em))
			if em != "" {
				adminEmails[em] = true
			}
		}
		log.Printf("Configured %d admin email(s)", len(adminEmails))
	}

	// JWT secret for local auth tokens
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
		if !devMode {
			log.Println("WARNING: JWT_SECRET not set, using default. Set JWT_SECRET for production.")
		}
	}

	// Email configuration for verification emails
	emailCfg := &email.Config{
		Host: os.Getenv("SMTP_HOST"),
		Port: os.Getenv("SMTP_PORT"),
		User: os.Getenv("SMTP_USER"),
		Pass: os.Getenv("SMTP_PASS"),
		From: os.Getenv("SMTP_FROM"),
	}
	if emailCfg.Port == "" {
		emailCfg.Port = "587"
	}
	if emailCfg.IsConfigured() {
		log.Printf("Email configured (SMTP: %s:%s)", emailCfg.Host, emailCfg.Port)
	} else {
		log.Println("Email not configured. Verification tokens will be logged to stdout.")
	}

	// App URL for verification links
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = allowedOrigin
	}

	h := handlers.New(s, emailCfg, jwtSecret, appURL)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Build middleware chain: CORS -> Auth -> routes
	corsHandler := middleware.CORS(allowedOrigin)(mux)

	// Wrap with auth middleware, but skip auth for OPTIONS requests
	authMiddleware := auth.Middleware(devMode, adminEmails, jwtSecret)
	authedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for preflight
		if r.Method == http.MethodOptions {
			corsHandler.ServeHTTP(w, r)
			return
		}
		authMiddleware(corsHandler).ServeHTTP(w, r)
	})

	// Apply CORS to the outer layer so preflight requests get CORS headers
	finalHandler := middleware.CORS(allowedOrigin)(authedHandler)

	if devMode {
		log.Println("DEV_MODE enabled - authentication disabled")
	}
	log.Printf("Server starting on :%s", port)
	fmt.Printf("Allowed CORS origin: %s\n", allowedOrigin)

	if err := http.ListenAndServe(":"+port, finalHandler); err != nil {
		log.Fatal(err)
	}
}
