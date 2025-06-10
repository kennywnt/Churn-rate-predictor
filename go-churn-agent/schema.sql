-- Ensure the uuid-ossp extension is enabled in your Supabase project.
-- You can check this under Database -> Extensions in the Supabase dashboard.
-- If not enabled, run: CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Create the customer_feedback table
CREATE TABLE public.customer_feedback (
    id UUID DEFAULT uuid_generate_v4() NOT NULL PRIMARY KEY,
    nls_score INT,
    feedback_text TEXT,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    comment_sentiment TEXT NULL,
    comment_topics TEXT[] NULL
);

-- Optional: Add a comment to describe the table
COMMENT ON TABLE public.customer_feedback IS 'Stores customer Net Promoter Score (NLS) and their feedback.';

-- 2. Create the churn_predictions table
CREATE TABLE public.churn_predictions (
    id UUID DEFAULT uuid_generate_v4() NOT NULL PRIMARY KEY,
    customer_feedback_id UUID REFERENCES public.customer_feedback(id) ON DELETE CASCADE, -- Ensures that if a feedback entry is deleted, its predictions are also deleted.
    churn_probability FLOAT,
    reason TEXT,
    predicted_at TIMESTAMPTZ DEFAULT now() NOT NULL
);

-- Optional: Add a comment to describe the table
COMMENT ON TABLE public.churn_predictions IS 'Stores churn predictions based on customer feedback.';

-- Optional: Add an index for faster lookups on the foreign key
CREATE INDEX idx_churn_predictions_customer_feedback_id ON public.churn_predictions(customer_feedback_id);
