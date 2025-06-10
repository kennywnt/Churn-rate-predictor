package main

import (
	"testing"
	"time"
)

const (
	reasonHighChurn   = "Low NLS score and/or negative feedback/sentiment."
	reasonModerateChurn = "Moderate NLS score or neutral feedback/sentiment."
	reasonLowChurn    = "High NLS score." // This one remains specific
)

// TestPredictChurn_LowNLSNegativeFeedback_KeywordDriven tests keyword-based high churn.
func TestPredictChurn_LowNLSNegativeFeedback_KeywordDriven(t *testing.T) {
	customerData := CustomerData{
		ID:       "test001",
		NLSScore: 4, // NLS < 5, but not < 3, to isolate keyword effect
		Feedback: "I am very unhappy with the terrible service.",
		CommentSentiment: "NEUTRAL", // Ensure sentiment isn't NEGATIVE to isolate keyword path
	}
	expectedProbability := 0.8
	expectedReason := reasonHighChurn

	prediction := PredictChurn(customerData)

	if prediction.ChurnProbability != expectedProbability {
		t.Errorf("Expected ChurnProbability %v, got %v for LowNLSNegativeFeedback_KeywordDriven", expectedProbability, prediction.ChurnProbability)
	}
	if prediction.Reason != expectedReason {
		t.Errorf("Expected Reason '%s', got '%s' for LowNLSNegativeFeedback_KeywordDriven", expectedReason, prediction.Reason)
	}
	if prediction.PredictedAt.IsZero() {
		t.Errorf("Expected PredictedAt to be set, but it was zero for LowNLSNegativeFeedback_KeywordDriven")
	}
}

// TestPredictChurn_HighNLS tests the scenario where NLS score is high.
func TestPredictChurn_HighNLS(t *testing.T) {
	customerData := CustomerData{
		ID:       "test002",
		NLSScore: 9,
		Feedback: "Excellent product, very happy!",
		CommentSentiment: "POSITIVE", // Sentiment shouldn't affect this path
	}
	expectedProbability := 0.1
	expectedReason := reasonLowChurn

	prediction := PredictChurn(customerData)

	if prediction.ChurnProbability != expectedProbability {
		t.Errorf("Expected ChurnProbability %v, got %v for HighNLS", expectedProbability, prediction.ChurnProbability)
	}
	if prediction.Reason != expectedReason {
		t.Errorf("Expected Reason '%s', got '%s' for HighNLS", expectedReason, prediction.Reason)
	}
	if prediction.PredictedAt.IsZero() {
		t.Errorf("Expected PredictedAt to be set, but it was zero for HighNLS")
	}
}

// TestPredictChurn_ModerateCase tests scenarios for moderate NLS score or neutral feedback.
func TestPredictChurn_ModerateCase(t *testing.T) {
	// Case 1: Moderate NLS score, neutral feedback, neutral sentiment
	customerData1 := CustomerData{
		ID:       "test003",
		NLSScore: 6,
		Feedback: "The service was okay.",
		CommentSentiment: "NEUTRAL",
	}
	expectedProbability := 0.4
	expectedReason := reasonModerateChurn

	prediction1 := PredictChurn(customerData1)

	if prediction1.ChurnProbability != expectedProbability {
		t.Errorf("Moderate Case 1: Expected ChurnProbability %v, got %v", expectedProbability, prediction1.ChurnProbability)
	}
	if prediction1.Reason != expectedReason {
		t.Errorf("Moderate Case 1: Expected Reason '%s', got '%s'", expectedReason, prediction1.Reason)
	}
	if prediction1.PredictedAt.IsZero() {
		t.Errorf("Moderate Case 1: Expected PredictedAt to be set, but it was zero")
	}

	// Case 2: NLS = 4 (low, but not < 3), no negative keywords, neutral sentiment
	customerData2 := CustomerData{
		ID:       "test004",
		NLSScore: 4,
		Feedback: "It's just okay.", // No negative keywords from the list
		CommentSentiment: "NEUTRAL",
	}
	prediction2 := PredictChurn(customerData2)

	if prediction2.ChurnProbability != expectedProbability {
		t.Errorf("Moderate Case 2: Expected ChurnProbability %v, got %v", expectedProbability, prediction2.ChurnProbability)
	}
	if prediction2.Reason != expectedReason {
		t.Errorf("Moderate Case 2: Expected Reason '%s', got '%s'", expectedReason, prediction2.Reason)
	}
	if prediction2.PredictedAt.IsZero() {
		t.Errorf("Moderate Case 2: Expected PredictedAt to be set, but it was zero")
	}
}

// TestPredictChurn_EdgeCaseLowNLSNegativeFeedback tests NLS score on the boundary (5) with negative feedback.
func TestPredictChurn_EdgeCaseLowNLSNegativeFeedback(t *testing.T) {
	customerData := CustomerData{
		ID:       "test005",
		NLSScore: 5,
		Feedback: "I am unhappy with the product.", // Has "unhappy"
		CommentSentiment: "NEGATIVE", // Even with negative sentiment, NLS=5 is not <3 or <5 with keyword for high churn.
	}
	// NLSScore is 5 (not < 5 for keyword rule, not < 3 for sentiment rule), so should be moderate.
	expectedProbability := 0.4
	expectedReason := reasonModerateChurn

	prediction := PredictChurn(customerData)

	if prediction.ChurnProbability != expectedProbability {
		t.Errorf("Expected ChurnProbability %v, got %v for EdgeCaseLowNLSNegativeFeedback", expectedProbability, prediction.ChurnProbability)
	}
	if prediction.Reason != expectedReason {
		t.Errorf("Expected Reason '%s', got '%s' for EdgeCaseLowNLSNegativeFeedback", expectedReason, prediction.Reason)
	}
	if prediction.PredictedAt.IsZero() {
		t.Errorf("Expected PredictedAt to be set, but it was zero for EdgeCaseLowNLSNegativeFeedback")
	}
}

// TestPredictChurn_NoFeedback tests behavior when feedback string is empty (implies no keywords, neutral sentiment if not explicitly set).
func TestPredictChurn_NoFeedback(t *testing.T) {
	customerData := CustomerData{
		ID:       "test006",
		NLSScore: 3, // Low score
		Feedback: "", // Empty feedback
		CommentSentiment: "NEUTRAL", // Assuming empty feedback means neutral sentiment from LLM or default
	}
	// With NLS=3, no keywords, and neutral sentiment, should be moderate.
	expectedProbability := 0.4
	expectedReason := reasonModerateChurn

	prediction := PredictChurn(customerData)

	if prediction.ChurnProbability != expectedProbability {
		t.Errorf("Expected ChurnProbability %v, got %v for NoFeedback", expectedProbability, prediction.ChurnProbability)
	}
	if prediction.Reason != expectedReason {
		t.Errorf("Expected Reason '%s', got '%s' for NoFeedback", expectedReason, prediction.Reason)
	}
	if prediction.PredictedAt.IsZero() {
		t.Errorf("Expected PredictedAt to be set, but it was zero for NoFeedback")
	}
}

// --- New Test Cases for Sentiment-Driven Logic ---

// TestPredictChurn_LowNLSLowSentiment tests high churn due to NLS < 3 and NEGATIVE sentiment.
func TestPredictChurn_LowNLSLowSentiment(t *testing.T) {
	customerData := CustomerData{
		ID:            "test007",
		NLSScore:      2, // NLS < 3
		Feedback:      "This is fine, whatever.", // No negative keywords
		CommentSentiment: "NEGATIVE",
	}
	expectedProbability := 0.8
	expectedReason := reasonHighChurn

	prediction := PredictChurn(customerData)

	if prediction.ChurnProbability != expectedProbability {
		t.Errorf("Expected ChurnProbability %v, got %v for LowNLSLowSentiment", expectedProbability, prediction.ChurnProbability)
	}
	if prediction.Reason != expectedReason {
		t.Errorf("Expected Reason '%s', got '%s' for LowNLSLowSentiment", expectedReason, prediction.Reason)
	}
	if prediction.PredictedAt.IsZero() {
		t.Errorf("Expected PredictedAt to be set for LowNLSLowSentiment")
	}
}

// TestPredictChurn_LowNLSNeutralSentimentKeywordDriven tests high churn by keywords when NLS < 5, even if sentiment is neutral.
func TestPredictChurn_LowNLSNeutralSentimentKeywordDriven(t *testing.T) {
	customerData := CustomerData{
		ID:            "test008",
		NLSScore:      4, // NLS < 5
		Feedback:      "This is really bad and poor.", // Contains negative keywords
		CommentSentiment: "NEUTRAL", // Sentiment is neutral, but keywords should trigger
	}
	expectedProbability := 0.8
	expectedReason := reasonHighChurn

	prediction := PredictChurn(customerData)

	if prediction.ChurnProbability != expectedProbability {
		t.Errorf("Expected ChurnProbability %v, got %v for LowNLSNeutralSentimentKeywordDriven", expectedProbability, prediction.ChurnProbability)
	}
	if prediction.Reason != expectedReason {
		t.Errorf("Expected Reason '%s', got '%s' for LowNLSNeutralSentimentKeywordDriven", expectedReason, prediction.Reason)
	}
	if prediction.PredictedAt.IsZero() {
		t.Errorf("Expected PredictedAt to be set for LowNLSNeutralSentimentKeywordDriven")
	}
}

// TestPredictChurn_VeryLowNLSNonNegativeSentimentNoKeywords tests moderate churn when NLS < 3 but sentiment is not NEGATIVE and no keywords.
func TestPredictChurn_VeryLowNLSNonNegativeSentimentNoKeywords(t *testing.T) {
	customerData := CustomerData{
		ID:            "test009",
		NLSScore:      2, // NLS < 3
		Feedback:      "It is okay, I guess.",     // No negative keywords
		CommentSentiment: "NEUTRAL",        // Sentiment is not NEGATIVE
	}
	expectedProbability := 0.4
	expectedReason := reasonModerateChurn

	prediction := PredictChurn(customerData)

	if prediction.ChurnProbability != expectedProbability {
		t.Errorf("Expected ChurnProbability %v, got %v for VeryLowNLSNonNegativeSentimentNoKeywords", expectedProbability, prediction.ChurnProbability)
	}
	if prediction.Reason != expectedReason {
		t.Errorf("Expected Reason '%s', got '%s' for VeryLowNLSNonNegativeSentimentNoKeywords", expectedReason, prediction.Reason)
	}
	if prediction.PredictedAt.IsZero() {
		t.Errorf("Expected PredictedAt to be set for VeryLowNLSNonNegativeSentimentNoKeywords")
	}

	// Also test with POSITIVE sentiment
	customerData.CommentSentiment = "POSITIVE"
	prediction = PredictChurn(customerData)
	if prediction.ChurnProbability != expectedProbability {
		t.Errorf("Expected ChurnProbability %v (POSITIVE sentiment), got %v", expectedProbability, prediction.ChurnProbability)
	}
	if prediction.Reason != expectedReason {
		t.Errorf("Expected Reason '%s' (POSITIVE sentiment), got '%s'", expectedReason, prediction.Reason)
	}
}

// TestPredictChurn_NLS4_NegativeSentiment_NoKeywords tests NLS=4 (not <3), Negative Sentiment, No Keywords -> Moderate
func TestPredictChurn_NLS4_NegativeSentiment_NoKeywords(t *testing.T) {
	customerData := CustomerData{
		ID:            "test010",
		NLSScore:      4,
		Feedback:      "Just a comment.", // No negative keywords
		CommentSentiment: "NEGATIVE",
	}
	// NLS is not < 3, so negative sentiment alone doesn't trigger high churn.
	// No negative keywords, so keyword rule doesn't trigger high churn.
	// Should be moderate.
	expectedProbability := 0.4
	expectedReason := reasonModerateChurn

	prediction := PredictChurn(customerData)

	if prediction.ChurnProbability != expectedProbability {
		t.Errorf("Expected ChurnProbability %v, got %v for NLS4_NegativeSentiment_NoKeywords", expectedProbability, prediction.ChurnProbability)
	}
	if prediction.Reason != expectedReason {
		t.Errorf("Expected Reason '%s', got '%s' for NLS4_NegativeSentiment_NoKeywords", expectedReason, prediction.Reason)
	}
	if prediction.PredictedAt.IsZero() {
		t.Errorf("Expected PredictedAt to be set for NLS4_NegativeSentiment_NoKeywords")
	}
}
