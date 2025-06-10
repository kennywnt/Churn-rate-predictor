Feature: Churn Prediction API

  Background:
    * url 'http://localhost:8080' # Base URL for the API - ensure the Go app runs on this port

  Scenario: Successful prediction with valid data
    Given path '/predict'
    And request { "nls_score": 8, "feedback_text": "Excellent service!" }
    When method post
    Then status 200
    And match response.customer_id == "#string"
    And match response.churn_probability == 0.1
    And match response.reason == "High NLS score."
    And match response.comment_sentiment == "#string"
    And match response.comment_topics == "#array"

  Scenario: Invalid NLS score (too high)
    Given path '/predict'
    And request { "nls_score": 11, "feedback_text": "Good." }
    When method post
    Then status 400
    And match response.error == "NLS score must be between 0 and 10"

  Scenario: Invalid NLS score (too low)
    Given path '/predict'
    And request { "nls_score": -1, "feedback_text": "Not good." }
    When method post
    Then status 400
    And match response.error == "NLS score must be between 0 and 10"

  Scenario: Missing NLS score (null)
    Given path '/predict'
    And request { "feedback_text": "Feedback only." } # nls_score field is completely missing
    When method post
    Then status 400
    And match response.error == "NLS score is required." # Matches Go code: if req.NLSScore == nil

  Scenario: Empty feedback text
    Given path '/predict'
    And request { "nls_score": 7, "feedback_text": "  " } # Test with whitespace only
    When method post
    Then status 400
    And match response.error == "Feedback text cannot be empty"

  Scenario: Malformed JSON request
    Given path '/predict'
    And request '{ "nls_score": 5, "feedback_text": "Test", }' # Extra comma makes it malformed
    When method post
    Then status 400
    And match response.error == "Invalid JSON request body."

  Scenario: Low NLS score and negative feedback
    Given path '/predict'
    And request { "nls_score": 2, "feedback_text": "Service was poor and I am unhappy." }
    When method post
    Then status 200
    And match response.customer_id == "#string"
    And match response.churn_probability == 0.8
    And match response.reason == "Low NLS score and/or negative feedback/sentiment." # Updated reason
    And match response.comment_sentiment == "#string"
    And match response.comment_topics == "#array"

  Scenario: Moderate NLS score
    Given path '/predict'
    And request { "nls_score": 6, "feedback_text": "It was alright." }
    When method post
    Then status 200
    And match response.customer_id == "#string"
    And match response.churn_probability == 0.4
    And match response.reason == "Moderate NLS score or neutral feedback/sentiment." # Updated reason
    And match response.comment_sentiment == "#string"
    And match response.comment_topics == "#array"
