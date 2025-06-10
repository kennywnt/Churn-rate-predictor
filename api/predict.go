package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync" // For once.Do

	"go-churn-agent/pkg/appcore" // Import the shared package
)

var initOnce sync.Once
var initErr error

// Initialize ensures that appcore clients are initialized only once.
func initialize() error {
	initOnce.Do(func() {
		log.Println("Attempting to initialize appcore clients...")
		initErr = appcore.InitClients()
		if initErr != nil {
			log.Printf("Error during appcore client initialization: %v", initErr)
		} else {
			log.Println("Appcore clients initialized successfully by handler.")
		}
	})
	return initErr
}

// PredictHandler is the entry point for Vercel.
func PredictHandler(w http.ResponseWriter, r *http.Request) {
	// Initialize appcore (Supabase client, etc.) safely for concurrent requests
	if err := initialize(); err != nil {
		// If init failed (e.g. missing env vars for Supabase), subsequent calls will also fail here.
		log.Printf("Initialization check failed: %v", err) // Log the specific init error
		appcore.RespondWithError(w, http.StatusInternalServerError, "Server initialization failed: "+err.Error())
		return
	}
    // Check if SupabaseClient is usable after initialization attempt
	if appcore.SupabaseClient == nil && initErr != nil {
		// This condition might be redundant if InitClients already fatally logs or returns clear error
		// but serves as an additional safeguard.
		log.Println("SupabaseClient is nil after initialization attempt, likely due to missing env vars for Supabase.")
		appcore.RespondWithError(w, http.StatusInternalServerError, "Supabase client not available due to initialization error.")
		return
	}


	log.Printf("Received request for /predict from %s", r.RemoteAddr)
	if r.Method != http.MethodPost {
		appcore.RespondWithError(w, http.StatusMethodNotAllowed, "Only POST method is allowed.")
		return
	}

	var req appcore.ApiPredictRequest // Use struct from appcore
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v", err)
		appcore.RespondWithError(w, http.StatusBadRequest, "Invalid JSON request body.")
		return
	}
	defer r.Body.Close()

	if req.NLSScore == nil {
		appcore.RespondWithError(w, http.StatusBadRequest, "NLS score is required.")
		return
	}
	if *req.NLSScore < 0 || *req.NLSScore > 10 {
		appcore.RespondWithError(w, http.StatusBadRequest, "NLS score must be between 0 and 10.")
		return
	}
	// Feedback text can be empty for LLM processing.

	log.Println("Fetching sentiment from Hugging Face...")
	sentiment, errSentiment := appcore.GetSentimentFromHF(req.FeedbackText)
	if errSentiment != nil {
		log.Printf("Warning: Could not get sentiment from Hugging Face: %v", errSentiment)
		sentiment = "UNKNOWN"
	}
	log.Printf("Sentiment received: %s", sentiment)

	candidateTopics := []string{"service", "product quality", "pricing", "customer support", "speed", "ease of use"}
	log.Println("Fetching topics from Hugging Face...")
	topics, errTopics := appcore.GetTopicsFromHF(req.FeedbackText, candidateTopics)
	if errTopics != nil {
		log.Printf("Warning: Could not get topics from Hugging Face: %v", errTopics)
	}
	log.Printf("Topics received: %v", topics)

	customerData := appcore.CustomerData{
		NLSScore:         *req.NLSScore,
		Feedback:         req.FeedbackText,
		CommentSentiment: sentiment,
		CommentTopics:    topics,
	}

	log.Println("Storing customer data (with insights) in Supabase...")
	customerID, err := appcore.StoreCustomerData(customerData) // Pass appcore.SupabaseClient implicitly
	if err != nil {
		log.Printf("Error storing customer data: %v", err)
		appcore.RespondWithError(w, http.StatusInternalServerError, "Failed to store customer data.")
		return
	}
	log.Printf("Customer data stored successfully. ID: %s\n", customerID)

	customerData.ID = customerID

	churnPrediction := appcore.PredictChurn(customerData)
	churnPrediction.CustomerID = customerID

	log.Println("Storing churn prediction in Supabase...")
	err = appcore.StoreChurnPrediction(churnPrediction) // Pass appcore.SupabaseClient implicitly
	if err != nil {
		log.Printf("Error storing churn prediction: %v", err)
		appcore.RespondWithError(w, http.StatusInternalServerError, "Failed to store churn prediction.")
		return
	}
	log.Println("Churn prediction stored successfully.")

	response := appcore.ApiResponse{ // Use struct from appcore
		CustomerID:       customerID,
		ChurnProbability: churnPrediction.ChurnProbability,
		Reason:           churnPrediction.Reason,
		CommentSentiment: customerData.CommentSentiment,
		CommentTopics:    customerData.CommentTopics,
	}
	appcore.RespondWithJSON(w, http.StatusOK, response)
}
