package main

import (
	"testing"
	"time" // Import time for PredictedAt comparison, if necessary, though not strictly for PredictChurn logic
)

// TestPredictChurn_LowNLSNegativeFeedback tests the scenario where NLS score is low and feedback is negative.
func TestPredictChurn_LowNLSNegativeFeedback(t *testing.T) {
	customerData := CustomerData{
		ID:       "test001",
		NLSScore: 2,
		Feedback: "I am very unhappy with the terrible service.",
	}
	expectedProbability := 0.8
	expectedReason := "Low NLS score and negative feedback."

	prediction := PredictChurn(customerData)

	if prediction.ChurnProbability != expectedProbability {
		t.Errorf("Expected ChurnProbability %v, got %v", expectedProbability, prediction.ChurnProbability)
	}
	if prediction.Reason != expectedReason {
		t.Errorf("Expected Reason '%s', got '%s'", expectedReason, prediction.Reason)
	}
	// Optional: Check if PredictedAt is set (not nil or zero time)
	if prediction.PredictedAt.IsZero() {
		t.Errorf("Expected PredictedAt to be set, but it was zero")
	}
}

// TestPredictChurn_HighNLS tests the scenario where NLS score is high.
func TestPredictChurn_HighNLS(t *testing.T) {
	customerData := CustomerData{
		ID:       "test002",
		NLSScore: 9,
		Feedback: "Excellent product, very happy!",
	}
	expectedProbability := 0.1
	expectedReason := "High NLS score."

	prediction := PredictChurn(customerData)

	if prediction.ChurnProbability != expectedProbability {
		t.Errorf("Expected ChurnProbability %v, got %v", expectedProbability, prediction.ChurnProbability)
	}
	if prediction.Reason != expectedReason {
		t.Errorf("Expected Reason '%s', got '%s'", expectedReason, prediction.Reason)
	}
	if prediction.PredictedAt.IsZero() {
		t.Errorf("Expected PredictedAt to be set, but it was zero")
	}
}

// TestPredictChurn_ModerateCase tests the scenario for moderate NLS score or neutral feedback.
func TestPredictChurn_ModerateCase(t *testing.T) {
	// Case 1: Moderate NLS score, neutral feedback
	customerData1 := CustomerData{
		ID:       "test003",
		NLSScore: 6,
		Feedback: "The service was okay.",
	}
	expectedProbability := 0.4
	expectedReason := "Moderate NLS score or neutral feedback."

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

	// Case 2: Low NLS score, but no strong negative keywords
	customerData2 := CustomerData{
		ID:       "test004",
		NLSScore: 3,
		Feedback: "It's not great, but not terrible either.", // "terrible" is a keyword, but let's assume logic needs exact match or more context
	}
	// This should still fall into moderate because "terrible" is a keyword, so the condition NLSScore < 5 AND hasNegativeFeedback would be true.
	// Let's adjust the feedback to ensure it's truly neutral for this test branch.
	customerData2.Feedback = "It's just okay." // No negative keywords from the list
	// Now it should be moderate.
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
		NLSScore: 5, // Edge case: NLSScore < 5 is the condition for high churn with negative feedback
		Feedback: "I am unhappy with the product.",
	}
	// Since NLSScore is 5 (not < 5), it should fall into the "Moderate NLS score or neutral feedback." category.
	expectedProbability := 0.4
	expectedReason := "Moderate NLS score or neutral feedback."

	prediction := PredictChurn(customerData)

	if prediction.ChurnProbability != expectedProbability {
		t.Errorf("Expected ChurnProbability %v, got %v", expectedProbability, prediction.ChurnProbability)
	}
	if prediction.Reason != expectedReason {
		t.Errorf("Expected Reason '%s', got '%s'", expectedReason, prediction.Reason)
	}
	if prediction.PredictedAt.IsZero() {
		t.Errorf("Expected PredictedAt to be set, but it was zero")
	}
}

// TestPredictChurn_NoFeedback tests behavior when feedback string is empty.
func TestPredictChurn_NoFeedback(t *testing.T) {
	customerData := CustomerData{
		ID:       "test006",
		NLSScore: 3, // Low score
		Feedback: "", // Empty feedback
	}
	// With low score and no negative keywords (empty feedback), should be moderate.
	expectedProbability := 0.4
	expectedReason := "Moderate NLS score or neutral feedback."

	prediction := PredictChurn(customerData)

	if prediction.ChurnProbability != expectedProbability {
		t.Errorf("Expected ChurnProbability %v, got %v", expectedProbability, prediction.ChurnProbability)
	}
	if prediction.Reason != expectedReason {
		t.Errorf("Expected Reason '%s', got '%s'", expectedReason, prediction.Reason)
	}
	if prediction.PredictedAt.IsZero() {
		t.Errorf("Expected PredictedAt to be set, but it was zero")
	}
}
