package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	supabase "github.com/supabase-community/supabase-go"
)

// --- Struct Definitions ---

// ApiPredictRequest defines the structure for the /predict endpoint request body.
type ApiPredictRequest struct {
	NLSScore     *int   `json:"nls_score"`
	FeedbackText string `json:"feedback_text"`
}

// ApiResponse defines the structure for successful /predict endpoint responses.
type ApiResponse struct {
	CustomerID       string   `json:"customer_id"`
	ChurnProbability float64  `json:"churn_probability"`
	Reason           string   `json:"reason"`
	CommentSentiment string   `json:"comment_sentiment,omitempty"` // New field
	CommentTopics    []string `json:"comment_topics,omitempty"`    // New field
}

// CustomerData represents the input data for a customer.
type CustomerData struct {
	ID               string    `json:"id,omitempty"`
	NLSScore         int       `json:"nls_score"`
	Feedback         string    `json:"feedback_text"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
	CommentSentiment string    `json:"comment_sentiment,omitempty"`
	CommentTopics    []string  `json:"comment_topics,omitempty"`
}

// ChurnPrediction represents the churn prediction output for a customer.
type ChurnPrediction struct {
	ID               string    `json:"id,omitempty"`
	CustomerID       string    `json:"customer_feedback_id"`
	ChurnProbability float64   `json:"churn_probability"`
	Reason           string    `json:"reason"`
	PredictedAt      time.Time `json:"predicted_at,omitempty"`
}

// HFSentimentRequest for sentiment analysis model.
type HFSentimentRequest struct {
	Inputs string `json:"inputs"`
}

// HFSentimentResponse structure for the sentiment model.
type HFSentimentResponse [][]struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

// HFZeroShotRequest for zero-shot classification model.
type HFZeroShotRequest struct {
	Inputs     string           `json:"inputs"`
	Parameters HFZeroShotParams `json:"parameters"`
}

// HFZeroShotParams holds parameters for the zero-shot model, like candidate labels.
type HFZeroShotParams struct {
	CandidateLabels []string `json:"candidate_labels"`
	MultiLabel      bool     `json:"multi_label"`
}

// HFZeroShotResponse structure for the zero-shot classification model.
type HFZeroShotResponse struct {
	Sequence string    `json:"sequence"`
	Labels   []string  `json:"labels"`
	Scores   []float64 `json:"scores"`
}

// --- Global Variables and Constants ---

var supabaseClient *supabase.Client

const (
	hfApiBaseURL        = "https://api-inference.huggingface.co/models/"
	sentimentModelID    = "distilbert-base-uncased-finetuned-sst-2-english"
	zeroShotModelID     = "facebook/bart-large-mnli"
	topicScoreThreshold = 0.8
)

// --- Helper Functions ---

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error during JSON marshalling"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func callHuggingFaceAPI(modelID string, requestBody interface{}) ([]byte, error) {
	hfToken := os.Getenv("HF_TOKEN")
	if hfToken == "" {
		return nil, fmt.Errorf("HF_TOKEN environment variable not set")
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request body for HF API: %w", err)
	}

	reqURL := hfApiBaseURL + modelID
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating new HTTP request for HF API to %s: %w", reqURL, err)
	}

	req.Header.Set("Authorization", "Bearer "+hfToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to Hugging Face API (%s): %w", reqURL, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body from HF API (%s): %w", reqURL, err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Hugging Face API (%s) returned non-200 status: %d. Response body: %s", reqURL, resp.StatusCode, string(bodyBytes))
		var hfError struct {
			Error         string   `json:"error"`
			EstimatedTime float64  `json:"estimated_time,omitempty"`
			Warnings      []string `json:"warnings,omitempty"`
		}
		if json.Unmarshal(bodyBytes, &hfError) == nil && hfError.Error != "" {
			if hfError.EstimatedTime > 0 {
				return nil, fmt.Errorf("HF API error for %s (model loading, try again in %.0fs): %s", modelID, hfError.EstimatedTime, hfError.Error)
			}
			return nil, fmt.Errorf("HF API error for %s: %s", modelID, hfError.Error)
		}
		return nil, fmt.Errorf("Hugging Face API (%s) request failed with status %d: %s", reqURL, resp.StatusCode, string(bodyBytes))
	}
	return bodyBytes, nil
}

func getSentimentFromHF(feedbackText string) (string, error) {
	if strings.TrimSpace(feedbackText) == "" {
		return "NEUTRAL", nil
	}
	requestPayload := HFSentimentRequest{Inputs: feedbackText}
	responseBody, err := callHuggingFaceAPI(sentimentModelID, requestPayload)
	if err != nil {
		return "UNKNOWN", fmt.Errorf("sentiment API call failed: %w", err)
	}

	var sentimentResponse HFSentimentResponse
	if err := json.Unmarshal(responseBody, &sentimentResponse); err != nil {
		log.Printf("Error unmarshalling sentiment response: %s. Body: %s", err, string(responseBody))
		return "UNKNOWN", fmt.Errorf("error unmarshalling sentiment response: %w", err)
	}

	if len(sentimentResponse) == 0 || len(sentimentResponse[0]) == 0 {
		log.Printf("Sentiment response format unexpected or empty. Body: %s", string(responseBody))
		return "UNKNOWN", fmt.Errorf("sentiment response format unexpected or empty")
	}

	highestScore := 0.0
	bestLabel := "NEUTRAL"
	for _, labelScorePair := range sentimentResponse[0] {
		if labelScorePair.Score > highestScore {
			highestScore = labelScorePair.Score
			bestLabel = labelScorePair.Label
		}
	}
	return bestLabel, nil
}

func getTopicsFromHF(feedbackText string, candidateTopics []string) ([]string, error) {
	if strings.TrimSpace(feedbackText) == "" || len(candidateTopics) == 0 {
		return []string{}, nil
	}

	requestPayload := HFZeroShotRequest{
		Inputs: feedbackText,
		Parameters: HFZeroShotParams{
			CandidateLabels: candidateTopics,
			MultiLabel:      true,
		},
	}
	responseBody, err := callHuggingFaceAPI(zeroShotModelID, requestPayload)
	if err != nil {
		return nil, fmt.Errorf("topic extraction API call failed: %w", err)
	}

	var zeroShotResponse HFZeroShotResponse
	if err := json.Unmarshal(responseBody, &zeroShotResponse); err != nil {
		log.Printf("Error unmarshalling zero-shot response: %s. Body: %s", err, string(responseBody))
		return nil, fmt.Errorf("error unmarshalling zero-shot response: %w", err)
	}

	var extractedTopics []string
	if len(zeroShotResponse.Labels) > 0 && len(zeroShotResponse.Scores) == len(zeroShotResponse.Labels) {
		for i, label := range zeroShotResponse.Labels {
			if zeroShotResponse.Scores[i] > topicScoreThreshold {
				extractedTopics = append(extractedTopics, label)
			}
		}
	} else {
		log.Printf("Zero-shot response format unexpected or empty. Body: %s", string(responseBody))
	}

	return extractedTopics, nil
}

// --- Business Logic Functions ---

func PredictChurn(data CustomerData) ChurnPrediction {
	prediction := ChurnPrediction{}

	negativeKeywords := []string{"bad", "poor", "terrible", "unhappy"}
	hasNegativeFeedback := false
	if data.Feedback != "" {
		for _, keyword := range negativeKeywords {
			if strings.Contains(strings.ToLower(data.Feedback), keyword) {
				hasNegativeFeedback = true
				break
			}
		}
	}

	isNegativeSentiment := strings.ToUpper(data.CommentSentiment) == "NEGATIVE"

	if (data.NLSScore < 5 && hasNegativeFeedback) || (data.NLSScore < 3 && isNegativeSentiment) {
		prediction.ChurnProbability = 0.8
		prediction.Reason = "Low NLS score and/or negative feedback/sentiment."
	} else if data.NLSScore >= 8 {
		prediction.ChurnProbability = 0.1
		prediction.Reason = "High NLS score."
	} else {
		prediction.ChurnProbability = 0.4
		prediction.Reason = "Moderate NLS score or neutral feedback/sentiment."
	}
	prediction.PredictedAt = time.Now()
	return prediction
}

func StoreCustomerData(client *supabase.Client, data CustomerData) (string, error) {
	var results []CustomerData
	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now()
	}
	rawData, count, err := client.From("customer_feedback").Insert(data, false, "", "", "").Execute()
	if err != nil {
		log.Printf("Raw error from Supabase: %#v\n", err)
		log.Printf("Type of error: %T\n", err)
		log.Printf("Count on error: %d\n", count)
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

func StoreChurnPrediction(client *supabase.Client, prediction ChurnPrediction) error {
	if prediction.PredictedAt.IsZero() {
		prediction.PredictedAt = time.Now()
	}
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

// --- HTTP Handler ---

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

	if req.NLSScore == nil {
		respondWithError(w, http.StatusBadRequest, "NLS score is required.")
		return
	}
	if *req.NLSScore < 0 || *req.NLSScore > 10 {
		respondWithError(w, http.StatusBadRequest, "NLS score must be between 0 and 10.")
		return
	}
	// Feedback text can be empty for LLM processing, so no strict non-empty check here
	// if strings.TrimSpace(req.FeedbackText) == "" {
	// respondWithError(w, http.StatusBadRequest, "Feedback text cannot be empty.")
	// return
	// }

	log.Println("Fetching sentiment from Hugging Face...")
	sentiment, errSentiment := getSentimentFromHF(req.FeedbackText)
	if errSentiment != nil {
		log.Printf("Warning: Could not get sentiment from Hugging Face: %v", errSentiment)
		sentiment = "UNKNOWN"
	}
	log.Printf("Sentiment received: %s", sentiment)

	candidateTopics := []string{"service", "product quality", "pricing", "customer support", "speed", "ease of use"}
	log.Println("Fetching topics from Hugging Face...")
	topics, errTopics := getTopicsFromHF(req.FeedbackText, candidateTopics)
	if errTopics != nil {
		log.Printf("Warning: Could not get topics from Hugging Face: %v", errTopics)
	}
	log.Printf("Topics received: %v", topics)

	customerData := CustomerData{
		NLSScore:         *req.NLSScore,
		Feedback:         req.FeedbackText,
		CommentSentiment: sentiment,
		CommentTopics:    topics,
	}

	log.Println("Storing customer data (with insights) in Supabase...")
	customerID, err := StoreCustomerData(supabaseClient, customerData)
	if err != nil {
		log.Printf("Error storing customer data: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to store customer data.")
		return
	}
	log.Printf("Customer data stored successfully. ID: %s\n", customerID)

	customerData.ID = customerID

	churnPrediction := PredictChurn(customerData)
	churnPrediction.CustomerID = customerID

	log.Println("Storing churn prediction in Supabase...")
	err = StoreChurnPrediction(supabaseClient, churnPrediction)
	if err != nil {
		log.Printf("Error storing churn prediction: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to store churn prediction.")
		return
	}
	log.Println("Churn prediction stored successfully.")

	response := ApiResponse{
		CustomerID:       customerID,
		ChurnProbability: churnPrediction.ChurnProbability,
		Reason:           churnPrediction.Reason,
		CommentSentiment: customerData.CommentSentiment, // Populate new field
		CommentTopics:    customerData.CommentTopics,    // Populate new field
	}
	respondWithJSON(w, http.StatusOK, response)
}

// --- Main Function ---

func main() {
	envSupabaseURL := os.Getenv("SUPABASE_URL")
	envSupabaseKey := os.Getenv("SUPABASE_KEY")
	hfToken := os.Getenv("HF_TOKEN")

	if envSupabaseURL == "" || envSupabaseKey == "" {
		log.Fatal("Error: SUPABASE_URL and SUPABASE_KEY environment variables must be set.")
	}
	if hfToken == "" {
		log.Fatal("Error: HF_TOKEN environment variable must be set for sentiment/topic analysis.")
	}

	var err error
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
