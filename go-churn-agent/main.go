package main

import (
	"log"
	// "go-churn-agent/pkg/appcore" // No longer needed here if main doesn't do much
	// "net/http" // No longer needed here
	// "os" // No longer needed here unless main itself needs env vars
)

// main is the original entry point. For Vercel, handlers in the /api directory
// will be used. This main function can be minimal or used for other purposes
// if the project were to also run as a traditional server.
// For a Vercel-only deployment focusing on serverless functions, this main.go
// might not even be strictly necessary if all handlers are in /api.
// However, Go modules often expect a main package.
func main() {
	log.Println("This is the root main.go. For Vercel deployment, handlers in the /api directory are used.")
	log.Println("If running locally for non-Vercel purposes, this main function would start the server.")
	log.Println("For Vercel, this main() func is not the primary entry point for API requests.")

	// Example: To run the server locally using the old mechanism (for testing outside Vercel)
	// you would re-add the http server setup here and call appcore.InitClients().
	// This is NOT how Vercel invokes the /api/predict.go handler.

	// --- Old server startup logic (now removed from here) ---
	// envSupabaseURL := os.Getenv("SUPABASE_URL")
	// envSupabaseKey := os.Getenv("SUPABASE_KEY")
	// hfToken := os.Getenv("HF_TOKEN")

	// if envSupabaseURL == "" || envSupabaseKey == "" {
	// 	log.Fatal("Error: SUPABASE_URL and SUPABASE_KEY environment variables must be set.")
	// }
	// if hfToken == "" {
	// 	log.Fatal("Error: HF_TOKEN environment variable must be set for sentiment/topic analysis.")
	// }

	// if err := appcore.InitClients(); err != nil { // Assuming InitClients is in appcore
	// 	log.Fatalf("Error initializing appcore clients: %v", err)
	// }
	// log.Println("Supabase client initialized successfully for local main.")

	// http.HandleFunc("/predict", predictHandler) // predictHandler would need to be defined or imported

	// port := ":8080"
	// log.Printf("Starting server on port %s via main.go (for local testing only)\n", port)
	// if err := http.ListenAndServe(port, nil); err != nil {
	// 	log.Fatalf("Failed to start server via main.go: %v", err)
	// }
}
