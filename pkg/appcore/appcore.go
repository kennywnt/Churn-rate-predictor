package appcore

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
// Note: This might be more appropriate in the `api` package if not used elsewhere in appcore.
// For now, keeping it here as it's related to CustomerData processing.
type ApiPredictRequest struct {
	NLSScore     *int   `json:"nls_score"`
	FeedbackText string `json:"feedback_text"`
}

// ApiResponse defines the structure for successful /predict endpoint responses.
// Similar to ApiPredictRequest, might be better in `api` if only used there.
type ApiResponse struct {
	CustomerID       string   `json:"customer_id"`
	ChurnProbability float64  `json:"churn_probability"`
	Reason           string   `json:"reason"`
	CommentSentiment string   `json:"comment_sentiment,omitempty"`
	CommentTopics    []string `json:"comment_topics,omitempty"`
}

type CustomerData struct {
	ID               string    `json:"id,omitempty"`
	NLSScore         int       `json:"nls_score"`
	Feedback         string    `json:"feedback_text"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
	CommentSentiment string    `json:"comment_sentiment,omitempty"`
	CommentTopics    []string  `json:"comment_topics,omitempty"`
}

type ChurnPrediction struct {
	ID               string    `json:"id,omitempty"`
	CustomerID       string    `json:"customer_feedback_id"`
	ChurnProbability float64   `json:"churn_probability"`
	Reason           string    `json:"reason"`
	PredictedAt      time.Time `json:"predicted_at,omitempty"`
}

type HFSentimentRequest struct {
	Inputs string `json:"inputs"`
}

type HFSentimentResponse [][]struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

type HFZeroShotRequest struct {
	Inputs     string           `json:"inputs"`
	Parameters HFZeroShotParams `json:"parameters"`
}

type HFZeroShotParams struct {
	CandidateLabels []string `json:"candidate_labels"`
	MultiLabel      bool     `json:"multi_label"`
}

type HFZeroShotResponse struct {
	Sequence string    `json:"sequence"`
	Labels   []string  `json:"labels"`
	Scores   []float64 `json:"scores"`
}

// --- Global Variables and Constants ---

// SupabaseClient needs to be initialized and set, e.g., by a main package or an Init function.
var SupabaseClient *supabase.Client

// HF API constants are public if needed by other packages, or keep them internal if only used here.
const (
	HfApiBaseURL        = "https://api-inference.huggingface.co/models/"
	SentimentModelID    = "distilbert-base-uncased-finetuned-sst-2-english"
	ZeroShotModelID     = "facebook/bart-large-mnli"
	TopicScoreThreshold = 0.8
)

// --- Helper Functions for HTTP responses (Exported) ---

func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, map[string]string{"error": message})
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
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

// --- Hugging Face API Functions (Exported) ---

func CallHuggingFaceAPI(modelID string, requestBody interface{}) ([]byte, error) {
	hfToken := os.Getenv("HF_TOKEN")
	if hfToken == "" {
		return nil, fmt.Errorf("HF_TOKEN environment variable not set")
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request body for HF API: %w", err)
	}

	reqURL := HfApiBaseURL + modelID
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

func GetSentimentFromHF(feedbackText string) (string, error) {
	if strings.TrimSpace(feedbackText) == "" {
		return "NEUTRAL", nil
	}
	requestPayload := HFSentimentRequest{Inputs: feedbackText}
	responseBody, err := CallHuggingFaceAPI(SentimentModelID, requestPayload)
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

func GetTopicsFromHF(feedbackText string, candidateTopics []string) ([]string, error) {
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
	responseBody, err := CallHuggingFaceAPI(ZeroShotModelID, requestPayload)
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
			if zeroShotResponse.Scores[i] > TopicScoreThreshold {
				extractedTopics = append(extractedTopics, label)
			}
		}
	} else {
		log.Printf("Zero-shot response format unexpected or empty. Body: %s", string(responseBody))
	}
	return extractedTopics, nil
}

// --- Business Logic Functions (Exported) ---

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

func StoreCustomerData(data CustomerData) (string, error) {
	if SupabaseClient == nil {
		return "", fmt.Errorf("SupabaseClient not initialized in appcore")
	}
	var results []CustomerData
	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now()
	}
	rawData, count, err := SupabaseClient.From("customer_feedback").Insert(data, false, "", "", "").Execute()
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

func StoreChurnPrediction(prediction ChurnPrediction) error {
	if SupabaseClient == nil {
		return fmt.Errorf("SupabaseClient not initialized in appcore")
	}
	if prediction.PredictedAt.IsZero() {
		prediction.PredictedAt = time.Now()
	}
	rawData, count, err := SupabaseClient.From("churn_predictions").Insert(prediction, false, "", "", "").Execute()
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

// InitClients initializes shared clients like Supabase.
// This should be called once from the main/handler package.
func InitClients() error {
	envSupabaseURL := os.Getenv("SUPABASE_URL")
	envSupabaseKey := os.Getenv("SUPABASE_KEY")
	hfToken := os.Getenv("HF_TOKEN") // Checked by callHuggingFaceAPI, but good to check early.

	if envSupabaseURL == "" || envSupabaseKey == "" {
		return fmt.Errorf("SUPABASE_URL and SUPABASE_KEY environment variables must be set")
	}
	if hfToken == "" {
		// This is checked within callHuggingFaceAPI, but an early check can be useful.
		// For Vercel, this might not cause a fatal startup if only some requests use HF.
		log.Println("Warning: HF_TOKEN environment variable not set. Sentiment/topic features will fail.")
	}

	var err error
	SupabaseClient, err = supabase.NewClient(envSupabaseURL, envSupabaseKey, nil)
	if err != nil {
		return fmt.Errorf("error initializing Supabase client: %w", err)
	}
	log.Println("Supabase client initialized successfully in appcore.")
	return nil
}
