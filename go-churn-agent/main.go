package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os" // Added for os.Getenv
	"strings"
	"time"

	supabase "github.com/supabase-community/supabase-go"
)

// ApiPredictRequest defines the structure for the /predict endpoint request body.
type ApiPredictRequest struct {
	NLSScore     *int   `json:"nls_score"` // Pointer to distinguish missing from 0
	FeedbackText string `json:"feedback_text"`
}

// ApiResponse defines the structure for successful /predict endpoint responses.
type ApiResponse struct {
	CustomerID       string  `json:"customer_id"`
	ChurnProbability float64 `json:"churn_probability"`
	Reason           string  `json:"reason"`
}

// Global Supabase client (or pass it around if preferred for larger apps)
var supabaseClient *supabase.Client

// CustomerData represents the input data for a customer.
type CustomerData struct {
	// ID is the unique identifier for the customer.
	ID string `json:"id,omitempty"`
	// NLSScore is the Net Promoter Score given by the customer.
	NLSScore int `json:"nls_score"`
	// Feedback is the textual feedback provided by the customer.
	Feedback string `json:"feedback_text"`
	// CreatedAt is the timestamp when the feedback was created.
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// ChurnPrediction represents the churn prediction output for a customer.
type ChurnPrediction struct {
	// ID is the unique identifier for the prediction record.
	ID string `json:"id,omitempty"`
	// CustomerID is the unique identifier for the customer feedback.
	CustomerID string `json:"customer_feedback_id"`
	// ChurnProbability is the calculated probability of the customer churning.
	ChurnProbability float64 `json:"churn_probability"`
	// Reason provides an explanation for the churn prediction.
	Reason string `json:"reason"`
	// PredictedAt is the timestamp when the prediction was made.
	PredictedAt time.Time `json:"predicted_at,omitempty"`
}

// respondWithError is a helper function to send JSON error responses.
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON is a helper function to send JSON responses.
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error during JSON marshalling"}`)) // fallback
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// PredictChurn predicts churn probability based on customer data.
// Note: The CustomerID in the returned ChurnPrediction will be empty here,
// as it's derived from the database record ID after storing CustomerData.
func PredictChurn(data CustomerData) ChurnPrediction {
	prediction := ChurnPrediction{} // CustomerID will be set after storing CustomerData

	negativeKeywords := []string{"bad", "poor", "terrible", "unhappy"}
	hasNegativeFeedback := false
	for _, keyword := range negativeKeywords {
		if strings.Contains(strings.ToLower(data.Feedback), keyword) {
			hasNegativeFeedback = true
			break
		}
	}

	if data.NLSScore < 5 && hasNegativeFeedback {
		prediction.ChurnProbability = 0.8
		prediction.Reason = "Low NLS score and negative feedback."
	} else if data.NLSScore >= 8 {
		prediction.ChurnProbability = 0.1
		prediction.Reason = "High NLS score."
	} else {
		prediction.ChurnProbability = 0.4
		prediction.Reason = "Moderate NLS score or neutral feedback."
	}
	prediction.PredictedAt = time.Now()
	return prediction
}

// StoreCustomerData inserts customer data into the 'customer_feedback' table.
// It returns the ID of the inserted row.
func StoreCustomerData(client *supabase.Client, data CustomerData) (string, error) {
	var results []CustomerData
	// Ensure CreatedAt is set if not already
	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now()
	}
	// The 'false' in Insert means we want the inserted row(s) back.
	rawData, count, err := client.From("customer_feedback").Insert(data, false, "", "", "").Execute()
	if err != nil {
		// More detailed error logging
		log.Printf("Raw error from Supabase: %#v\n", err)
		log.Printf("Type of error: %T\n", err)
		log.Printf("Count on error: %d\n", count) // Log the count
		// Also print rawData if err is not nil, as it might contain the actual error response
		if len(rawData) > 0 {
			log.Printf("Raw response data on error: %s\n", string(rawData))
		}
		return "", fmt.Errorf("error storing customer data (count: %d): %w", count, err)
	}
	if err := json.Unmarshal(rawData, &results); err != nil {
		return "", fmt.Errorf("error unmarshalling customer data: %w", err)
	}
	if len(results) == 0 {
		return "", fmt.Errorf("no data returned after insert")
	}
	return results[0].ID, nil
}

// StoreChurnPrediction inserts a churn prediction into the 'churn_predictions' table.
func StoreChurnPrediction(client *supabase.Client, prediction ChurnPrediction) error {
	// Ensure PredictedAt is set if not already
	if prediction.PredictedAt.IsZero() {
		prediction.PredictedAt = time.Now()
	}
	// The 'false' in Insert means we want the inserted row(s) back.
	// Execute() returns (responseData, count, error).
	rawData, count, err := client.From("churn_predictions").Insert(prediction, false, "", "", "").Execute()
	if err != nil {
		log.Printf("Raw error from Supabase (prediction): %#v\n", err)
		log.Printf("Type of error (prediction): %T\n", err)
		log.Printf("Count on error (prediction): %d\n", count)
		if len(rawData) > 0 {
			log.Printf("Raw response data on error (prediction): %s\n", string(rawData))
		}
		return fmt.Errorf("error storing churn prediction (count: %d): %w", count, err)
	}
	return nil
}

func predictHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request for /predict from %s", r.RemoteAddr)
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Only POST method is allowed.")
		return
	}

	var req ApiPredictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid JSON request body.")
		return
	}
	defer r.Body.Close()

	// Validate input
	if req.NLSScore == nil {
		respondWithError(w, http.StatusBadRequest, "NLS score is required.")
		return
	}
	if *req.NLSScore < 0 || *req.NLSScore > 10 {
		respondWithError(w, http.StatusBadRequest, "NLS score must be between 0 and 10.")
		return
	}
	// Basic validation for feedback text, can be expanded
	if strings.TrimSpace(req.FeedbackText) == "" {
		respondWithError(w, http.StatusBadRequest, "Feedback text cannot be empty.")
		return
	}

	// Prepare CustomerData
	customerData := CustomerData{
		NLSScore: *req.NLSScore,
		Feedback: req.FeedbackText,
	}

	// Store Customer Data
	log.Println("Storing customer data in Supabase...")
	customerID, err := StoreCustomerData(supabaseClient, customerData)
	if err != nil {
		log.Printf("Error storing customer data: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to store customer data.")
		return
	}
	log.Printf("Customer data stored successfully. ID: %s\n", customerID)

	// Predict Churn
	churnPrediction := PredictChurn(customerData)
	churnPrediction.CustomerID = customerID // Set the FK

	// Store Churn Prediction
	log.Println("Storing churn prediction in Supabase...")
	err = StoreChurnPrediction(supabaseClient, churnPrediction)
	if err != nil {
		log.Printf("Error storing churn prediction: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to store churn prediction.")
		return
	}
	log.Println("Churn prediction stored successfully.")

	// Respond
	response := ApiResponse{
		CustomerID:       customerID,
		ChurnProbability: churnPrediction.ChurnProbability,
		Reason:           churnPrediction.Reason,
	}
	respondWithJSON(w, http.StatusOK, response)
}

func main() {
	envSupabaseURL := os.Getenv("SUPABASE_URL")
	envSupabaseKey := os.Getenv("SUPABASE_KEY")

	if envSupabaseURL == "" || envSupabaseKey == "" {
		log.Fatal("Error: SUPABASE_URL and SUPABASE_KEY environment variables must be set.")
	}

	var err error // Declare err here to be used by NewClient
	supabaseClient, err = supabase.NewClient(envSupabaseURL, envSupabaseKey, nil)
	if err != nil {
		log.Fatalf("Error initializing Supabase client: %v", err)
	}
	log.Println("Supabase client initialized successfully.")

	http.HandleFunc("/predict", predictHandler)

	port := ":8080"
	log.Printf("Starting server on port %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
