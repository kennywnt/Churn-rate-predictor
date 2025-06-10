# Go Churn Prediction Agent

## Overview

The Go Churn Prediction Agent is a **REST API server** that predicts customer churn probability based on their Net Promoter Score (NLS) and textual feedback. It interfaces with a Supabase backend to store customer data and the generated churn predictions. The agent uses a rule-based algorithm for its predictions.

## Features

-   Provides a REST API endpoint (`/predict`) for churn prediction.
-   Enriches customer feedback with AI-driven sentiment analysis and topic extraction using Hugging Face models.
-   Predicts churn probability using NLS score, keyword-based feedback analysis, and comment sentiment.
-   Stores customer feedback data (including LLM insights) and churn predictions in a Supabase database.
-   Configuration via environment variables for Supabase and Hugging Face credentials.
-   Dockerized for easy setup and deployment.
-   Includes basic unit tests for the prediction logic.
-   Includes API tests using Karate.

## Prerequisites

-   **Go:** Version 1.22 or newer.
-   **Supabase:** An active Supabase account and a project.
-   **Java Development Kit (JDK):** Version 11 or newer (for Karate tests).
-   **Maven:** Version 3.6+ (for Karate tests).
-   **Docker:** Required if you intend to run the application using Docker.

## Setup & Configuration

### 1. Supabase Setup

1.  **Create a Supabase Project:**
    *   Go to [Supabase](https://supabase.com/) and create a new project if you haven't already.
2.  **Database Schema:**
    *   The required database tables can be created using the schema provided in the `schema.sql` file in this repository. You can run this SQL in the Supabase SQL Editor for your project.
    *   Ensure the `uuid-ossp` extension is enabled in your Supabase project (Database -> Extensions). If not, run: `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";` before running `schema.sql`.
3.  **Get Project Credentials:**
    *   **Project URL:** Find this in your Supabase project settings under "API" -> "Project URL".
    *   **Service Role Key:** Find this under "API" -> "Project API Keys" -> "service_role" (secret). **Important:** Keep this key confidential.

### 2. Application Configuration

The application requires the following environment variables to be set:

-   `SUPABASE_URL`: Your Supabase project's API URL.
-   `SUPABASE_KEY`: Your Supabase project's Service Role Key (secret).
-   `HF_TOKEN`: Your Hugging Face API token (for accessing sentiment and topic models). You can get a token from [Hugging Face](https://huggingface.co/settings/tokens).

**Example of setting environment variables (Linux/macOS):**
```bash
export SUPABASE_URL="https://your-project-id.supabase.co"
export SUPABASE_KEY="your-very-long-service-role-key"
export HF_TOKEN="your_hugging_face_api_token"
```
**Example (Windows PowerShell):**
```powershell
$Env:SUPABASE_URL="https://your-project-id.supabase.co"
$Env:SUPABASE_KEY="your-very-long-service-role-key"
$Env:HF_TOKEN="your_hugging_face_api_token"
```
The application will fail to start if these environment variables are not set.

## Building and Running Locally (API Server)

1.  **Clone the repository (if you haven't already):**
    ```bash
    git clone <repository_url>
    cd <repository_name>/go-churn-agent
    ```
2.  **Set Supabase Environment Variables:**
    Ensure you have set the `SUPABASE_URL` and `SUPABASE_KEY` environment variables as described above.
3.  **Run the application (API Server):**
    ```bash
    go run main.go
    ```
    This will start the API server, typically listening on port `8080`. You should see a log message like "Starting server on port :8080".
4.  **Run Go unit tests:**
    ```bash
    go test -v ./...
    ```

## Running API Tests with Karate

The API tests are written using Karate and managed with Maven.

1.  **Start the Go API Server:**
    Ensure the `go-churn-agent` API server is running locally (see "Building and Running Locally" section). By default, it should be accessible at `http://localhost:8080`.
2.  **Navigate to the Karate tests directory:**
    ```bash
    cd karate-tests
    ```
3.  **Run the Karate tests using Maven:**
    ```bash
    mvn test
    ```
    Test reports are typically generated in the `karate-tests/target/surefire-reports` directory. You can find an HTML report (e.g., `karate-summary.html`) in `karate-tests/target/karate-reports/`.

## Building and Running with Docker

1.  **Build the Docker image:**
    From within the `go-churn-agent` root directory:
    ```bash
    docker build -t go-churn-agent .
    ```
2.  **Run the Docker container:**
    You must provide the Supabase and Hugging Face credentials as environment variables to the container.
    ```bash
    docker run -p 8080:8080 --rm \
      -e SUPABASE_URL="your_actual_supabase_url" \
      -e SUPABASE_KEY="your_actual_supabase_key" \
      -e HF_TOKEN="your_actual_hf_token" \
      go-churn-agent
    ```
    *   `-p 8080:8080`: Maps port 8080 from the container to port 8080 on your host machine.
    *   `--rm`: Automatically removes the container when it exits.
    *   `-e SUPABASE_URL=...`: Sets the Supabase URL environment variable inside the container.
    *   `-e SUPABASE_KEY=...`: Sets the Supabase Key environment variable inside the container.

## API Documentation

The application provides a single REST API endpoint for churn prediction.

### Endpoint: `POST /predict`

*   **Description:** Receives customer NLS score and feedback text. It then:
    1.  Stores the customer feedback data in Supabase.
    2.  Predicts churn based on the input.
    3.  Stores the churn prediction in Supabase.
    4.  Returns the `customer_id` (from stored feedback), `churn_probability`, and `reason`.
*   **Request Body (JSON):**
    ```json
    {
      "nls_score": 8,
      "feedback_text": "Great service, very happy!"
    }
    ```
    *   `nls_score` (integer, required): Net Promoter Score, must be between 0 and 10.
    *   `feedback_text` (string, required): Customer's textual feedback, cannot be empty.

*   **Success Response (`200 OK`) (JSON):**
    ```json
    {
      "customer_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "churn_probability": 0.8,
      "reason": "Low NLS score and/or negative feedback/sentiment.",
      "comment_sentiment": "NEGATIVE",
      "comment_topics": ["customer service", "wait times"]
    }
    ```
    *   `comment_sentiment` (string, optional): The sentiment derived from the feedback text (e.g., "POSITIVE", "NEGATIVE", "NEUTRAL", "UNKNOWN").
    *   `comment_topics` (array of strings, optional): A list of topics extracted from the feedback text.

*   **Error Responses (JSON):**
    *   **`400 Bad Request`**: Sent for issues like invalid JSON, missing required fields, or invalid data values (e.g., NLS score out of range).
        Example:
        ```json
        { "error": "NLS score must be between 0 and 10" }
        ```
        ```json
        { "error": "Feedback text cannot be empty" }
        ```
        ```json
        { "error": "Invalid JSON request body." }
        ```
    *   **`405 Method Not Allowed`**: If the request method is not `POST`.
        Example:
        ```json
        { "error": "Only POST method is allowed." }
        ```
    *   **`500 Internal Server Error`**: For server-side issues, such as failure to communicate with Supabase or other unexpected errors.
        Example:
        ```json
        { "error": "Failed to store customer data." }
        ```

## Project Structure

```
go-churn-agent/
├── Dockerfile          # Instructions for building the Docker image
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── main.go             # Main application logic (REST API server), Supabase interaction
├── main_test.go        # Go unit tests for PredictChurn function
├── README.md           # This file
├── schema.sql          # SQL schema for Supabase tables
└── karate-tests/       # Karate API tests
    ├── pom.xml         # Maven configuration for Karate tests
    └── src/
        └── test/
            └── java/
                └── com/
                    └── example/
                        └── api/
                            ├── PredictApiRunner.java  # Karate test runner
                            └── predict.feature        # Karate feature file for API tests
```

## Note on Prediction Logic Evolution
The integration of LLM-derived insights (sentiment and topics) into the churn prediction logic is iterative.
- Currently, `CommentSentiment` is used in conjunction with NLS scores to refine churn probability (e.g., a very low NLS score combined with negative sentiment strongly indicates high churn).
- `CommentTopics` are collected and stored but are not yet directly used to alter the churn probability score in this phase. They are available for data analysis and can be incorporated into more advanced prediction models in future iterations.
