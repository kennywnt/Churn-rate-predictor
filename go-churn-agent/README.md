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
    *   Go to [Supabase](https://supabase.com/) and create a new project.
2.  **Database Schema:**
    *   Use the schema in `schema.sql` to create the necessary tables (`customer_feedback`, `churn_predictions`) in your Supabase project via the SQL Editor.
    *   Ensure the `uuid-ossp` extension is enabled in Supabase (Database -> Extensions). If not, run: `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";` before applying the schema.
3.  **Get Project Credentials:**
    *   **Project URL:** Found in Supabase project settings (API -> Project URL).
    *   **Service Role Key:** Found in Supabase project settings (API -> Project API Keys -> `service_role` secret). Keep this confidential.

### 2. Hugging Face API Token
    *   You'll need an API token from Hugging Face to use the sentiment analysis and topic extraction models.
    *   Obtain a token from [Hugging Face User Access Tokens](https://huggingface.co/settings/tokens).

### 3. Environment Variables

The application requires the following environment variables to be set:

-   `SUPABASE_URL`: Your Supabase project's API URL.
-   `SUPABASE_KEY`: Your Supabase project's Service Role Key.
-   `HF_TOKEN`: Your Hugging Face API token.

The application will fail to start if these are not correctly configured. For local development with Vercel CLI, these can be placed in a `.env` file. For Vercel deployments, set them in the project's environment variable settings on the Vercel dashboard. For Docker, pass them during `docker run`.

## Local Development with Vercel CLI

This is the recommended way to run the application locally as it closely mimics the Vercel deployment environment.

1.  **Install Vercel CLI:**
    ```bash
    npm install -g vercel
    ```
2.  **Set Environment Variables:**
    *   Create a `.env` file in the project root (`go-churn-agent/`) with your credentials:
        ```env
        SUPABASE_URL=your_supabase_url
        SUPABASE_KEY=your_supabase_service_role_key
        HF_TOKEN=your_hugging_face_token
        ```
        **Important:** Add `.env` to your `.gitignore` file to prevent committing secrets.
    *   Alternatively, link your project to Vercel (`vercel link`) and manage environment variables through the Vercel dashboard.
3.  **Run the Development Server:**
    From the `go-churn-agent` root directory:
    ```bash
    vercel dev
    ```
    The Vercel CLI will typically start the server on `http://localhost:3000`. The API endpoint `/predict` will be available (e.g., `http://localhost:3000/predict`, due to routing in `vercel.json`).

## Deploying to Vercel

1.  **Sign up/Log in to Vercel.**
2.  **Install Vercel CLI** (if not already done): `npm install -g vercel`.
3.  **Connect your Git Repository:**
    *   Import your project into Vercel by connecting it to your GitHub, GitLab, or Bitbucket repository.
4.  **Configure Environment Variables:**
    *   In your Vercel project settings (Dashboard -> Project -> Settings -> Environment Variables), add `SUPABASE_URL`, `SUPABASE_KEY`, and `HF_TOKEN` with their respective values.
5.  **Deploy:**
    *   Vercel automatically builds and deploys your project when you push to the connected Git branch (e.g., `main`).
    *   The `vercel.json` file in the repository root configures Vercel to build `api/predict.go` as a serverless function and route `POST /predict` requests to it.
6.  **Access Your Deployment:**
    *   Vercel will provide a production URL for your deployment.

## Building and Running with Docker (Alternative Deployment / Testing)

This method builds a standalone Docker container running a traditional Go HTTP server. It uses the `cmd/server/main.go` entrypoint.

1.  **Build the Docker image:**
    From the `go-churn-agent` root directory:
    ```bash
    docker build -t go-churn-agent .
    ```
2.  **Run the Docker container:**
    Provide the necessary environment variables:
    ```bash
    docker run -p 8080:8080 --rm \
      -e SUPABASE_URL="your_actual_supabase_url" \
      -e SUPABASE_KEY="your_actual_supabase_key" \
      -e HF_TOKEN="your_actual_hf_token" \
      go-churn-agent
    ```
    *   `-p 8080:8080`: Maps port 8080 from the container to port 8080 on your host. The API will be accessible at `http://localhost:8080/predict`.
    *   `--rm`: Automatically removes the container when it exits.

## Go Modules and Dependencies
If you modify dependencies in `go.mod` (e.g., by adding new packages in `pkg/appcore` or `api`), run:
```bash
go mod tidy
```
This will ensure `go.sum` is updated and dependencies are correctly managed.

## Running Tests

### Go Unit Tests
These test the core Go logic in `pkg/appcore`.
```bash
go test -v ./...
```

### API Tests with Karate
These tests target a running instance of the API.

1.  **Start the API Server:**
    *   **For `vercel dev`:** Run `vercel dev` (targets `http://localhost:3000` by default in `predict.feature`).
    *   **For Docker:** Run the Docker container as described above (tests would need URL in `predict.feature` changed to `http://localhost:8080`).
    *   **For deployed Vercel instance:** Change the URL in `predict.feature` to your Vercel deployment URL.
2.  **Run Karate Tests:**
    Navigate to the `karate-tests` directory:
    ```bash
    cd karate-tests
    mvn test
    ```
    Or, from the project root:
    ```bash
    mvn test -f karate-tests/pom.xml
    ```
    Test reports are generated in `karate-tests/target/surefire-reports` and `karate-tests/target/karate-reports/`.

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
├── api/
│   └── predict.go      # Vercel serverless function handler for /predict
├── cmd/
│   └── server/
│       └── main.go     # Entrypoint for standalone Docker server
├── pkg/
│   └── appcore/
│       └── appcore.go  # Shared core logic, types, client initializations
├── karate-tests/       # Karate API tests
│   ├── pom.xml         # Maven configuration for Karate tests
│   └── src/test/java/com/example/api/
│       ├── PredictApiRunner.java # Karate test runner
│       └── predict.feature       # Karate feature file
├── .env.example        # Example environment file (recommend creating a .env based on this)
├── Dockerfile          # For building a standalone Docker image
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── main.go             # Minimal main, primarily for Go module structure
├── main_test.go        # Go unit tests for pkg/appcore logic
├── README.md           # This file
├── schema.sql          # SQL schema for Supabase tables
└── vercel.json         # Vercel deployment configuration
```

## Note on Prediction Logic Evolution
The integration of LLM-derived insights (sentiment and topics) into the churn prediction logic is iterative.
- Currently, `CommentSentiment` is used in conjunction with NLS scores to refine churn probability (e.g., a very low NLS score combined with negative sentiment strongly indicates high churn).
- `CommentTopics` are collected and stored but are not yet directly used to alter the churn probability score in this phase. They are available for data analysis and can be incorporated into more advanced prediction models in future iterations.
