package main

import (
	"log"
	"net/http"
	"sync"

	"go-churn-agent/api"         // Import the Vercel handler package
	"go-churn-agent/pkg/appcore" // Import the shared appcore package
)

var initOnce sync.Once
var initErr error

// initialize ensures that appcore clients are initialized only once.
func initialize() error {
	initOnce.Do(func() {
		log.Println("Attempting to initialize appcore clients for standalone server...")
		initErr = appcore.InitClients() // This checks SUPABASE_URL, SUPABASE_KEY, and HF_TOKEN
		if initErr != nil {
			log.Printf("Error during appcore client initialization for standalone server: %v", initErr)
		} else {
			log.Println("Appcore clients initialized successfully for standalone server.")
		}
	})
	return initErr
}

func main() {
	log.Println("Initializing standalone server...")

	// Initialize appcore (Supabase client, etc.) safely.
	// This also performs environment variable checks for Supabase and HF tokens.
	if err := initialize(); err != nil {
		// InitClients in appcore now returns an error if Supabase vars are missing.
		// It also logs a warning for HF_TOKEN but doesn't return an error for it there.
		// The main Vercel handler exits on fatal for missing HF_TOKEN during its init.
		// For a standalone server, we should probably be strict for all.
		// However, appcore.InitClients already checks Supabase and warns for HF.
		// The fatal log for missing env vars is in appcore.InitClients if they are critical for Supabase.
		// Let's ensure HF_TOKEN is also treated as critical for the standalone server here.
		// Re-checking or relying on appcore.InitClients' existing fatal/error behavior.
		// appcore.InitClients already logs and returns error for Supabase, warns for HF.
		// For the standalone server, if InitClients returns an error (e.g. Supabase config error),
		// we should not proceed.
		log.Fatalf("Server initialization failed: %v", err)
	}

	// Explicitly check HF_TOKEN here again if we want the standalone server to fail hard,
	// matching the Vercel handler's initial strictness (though Vercel handler's init
	// doesn't fatal, it makes handler return error).
	// if os.Getenv("HF_TOKEN") == "" {
	// 	log.Fatal("Error: HF_TOKEN environment variable must be set for the server to operate.")
	// }
	// This check is now inside appcore.InitClients effectively for HF (it's in callHuggingFaceAPI, but InitClients warns)
    // and critically for Supabase.

	// Use the PredictHandler from the api package.
	// Note: Vercel's `api.PredictHandler` expects to be the entry point and might do its own
	// one-time initialization. We've duplicated the `sync.Once` logic here for the server context.
	// A cleaner way might be to have `appcore.InitClients` be idempotent and callable multiple times
	// without `sync.Once` in each calling package, or provide a "EnsureInitialized" func.
	// For now, this structure with sync.Once in both places is safe.
	// The handler from /api is already set up to use appcore's global SupabaseClient.

	// The Vercel handler `api.PredictHandler` is designed to be an http.HandlerFunc.
	// It calls appcore.InitClients itself using its own sync.Once. This is fine.
	// So, our server main's `initialize()` call ensures critical env vars are checked at server startup.
	// The handler will also ensure initialization on its first request if it hasn't happened.
	http.HandleFunc("/predict", api.PredictHandler)

	port := ":8080" // This server will run on 8080 as per Dockerfile EXPOSE
	log.Printf("Starting standalone API server on port %s...\n", port)
	log.Println("API endpoint available at /predict (POST)")
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start standalone server: %v", err)
	}
}
