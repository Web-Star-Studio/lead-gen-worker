-- Migration: Create usage_metrics table for tracking AI usage and costs
-- Description: Stores detailed metrics for each AI operation including tokens, costs, and performance

-- Create operation_type enum
DO $$ BEGIN
    CREATE TYPE operation_type AS ENUM (
        'data_extraction',
        'pre_call_report',
        'cold_email',
        'website_scraping'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create usage_metrics table
CREATE TABLE IF NOT EXISTS usage_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    job_id UUID REFERENCES jobs(id) ON DELETE SET NULL,
    lead_id UUID REFERENCES leads(id) ON DELETE SET NULL,
    operation_type operation_type NOT NULL,
    model TEXT NOT NULL,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    estimated_cost_usd DECIMAL(10, 6) NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_usage_metrics_user_id ON usage_metrics(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_metrics_created_at ON usage_metrics(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_metrics_operation_type ON usage_metrics(operation_type);
CREATE INDEX IF NOT EXISTS idx_usage_metrics_user_created ON usage_metrics(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_usage_metrics_job_id ON usage_metrics(job_id);
CREATE INDEX IF NOT EXISTS idx_usage_metrics_lead_id ON usage_metrics(lead_id);

-- Enable RLS
ALTER TABLE usage_metrics ENABLE ROW LEVEL SECURITY;

-- RLS Policies

-- Users can view their own usage metrics
CREATE POLICY "Users can view own usage metrics"
    ON usage_metrics
    FOR SELECT
    USING (auth.uid() = user_id);

-- Service role can insert usage metrics (for the worker)
CREATE POLICY "Service role can insert usage metrics"
    ON usage_metrics
    FOR INSERT
    WITH CHECK (true);

-- Service role can view all metrics (for admin dashboards)
CREATE POLICY "Service role can view all metrics"
    ON usage_metrics
    FOR SELECT
    USING (auth.role() = 'service_role');

-- Grant permissions
GRANT SELECT ON usage_metrics TO authenticated;
GRANT INSERT ON usage_metrics TO service_role;
GRANT SELECT ON usage_metrics TO service_role;

-- Create a view for aggregated daily usage (useful for dashboards)
CREATE OR REPLACE VIEW daily_usage_summary AS
SELECT 
    user_id,
    DATE(created_at) as date,
    operation_type,
    COUNT(*) as total_calls,
    SUM(CASE WHEN success THEN 1 ELSE 0 END) as successful_calls,
    SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failed_calls,
    SUM(input_tokens) as total_input_tokens,
    SUM(output_tokens) as total_output_tokens,
    SUM(total_tokens) as total_tokens,
    SUM(estimated_cost_usd) as total_cost_usd,
    AVG(duration_ms) as avg_duration_ms
FROM usage_metrics
GROUP BY user_id, DATE(created_at), operation_type;

-- Grant access to the view
GRANT SELECT ON daily_usage_summary TO authenticated;
GRANT SELECT ON daily_usage_summary TO service_role;

-- Create a function to get usage summary for a user within a date range
CREATE OR REPLACE FUNCTION get_user_usage_summary(
    p_user_id UUID,
    p_start_date TIMESTAMPTZ DEFAULT NULL,
    p_end_date TIMESTAMPTZ DEFAULT NULL
)
RETURNS TABLE (
    total_calls BIGINT,
    successful_calls BIGINT,
    failed_calls BIGINT,
    success_rate DECIMAL,
    total_input_tokens BIGINT,
    total_output_tokens BIGINT,
    total_tokens BIGINT,
    total_cost_usd DECIMAL,
    avg_cost_per_call DECIMAL,
    avg_tokens_per_call DECIMAL,
    avg_duration_ms DECIMAL
) 
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::BIGINT as total_calls,
        SUM(CASE WHEN um.success THEN 1 ELSE 0 END)::BIGINT as successful_calls,
        SUM(CASE WHEN NOT um.success THEN 1 ELSE 0 END)::BIGINT as failed_calls,
        CASE 
            WHEN COUNT(*) > 0 
            THEN ROUND((SUM(CASE WHEN um.success THEN 1 ELSE 0 END)::DECIMAL / COUNT(*)) * 100, 2)
            ELSE 0 
        END as success_rate,
        COALESCE(SUM(um.input_tokens), 0)::BIGINT as total_input_tokens,
        COALESCE(SUM(um.output_tokens), 0)::BIGINT as total_output_tokens,
        COALESCE(SUM(um.total_tokens), 0)::BIGINT as total_tokens,
        COALESCE(SUM(um.estimated_cost_usd), 0)::DECIMAL as total_cost_usd,
        CASE 
            WHEN COUNT(*) > 0 
            THEN ROUND(SUM(um.estimated_cost_usd) / COUNT(*), 6)
            ELSE 0 
        END as avg_cost_per_call,
        CASE 
            WHEN COUNT(*) > 0 
            THEN ROUND(SUM(um.total_tokens)::DECIMAL / COUNT(*), 2)
            ELSE 0 
        END as avg_tokens_per_call,
        CASE 
            WHEN COUNT(*) > 0 
            THEN ROUND(AVG(um.duration_ms), 2)
            ELSE 0 
        END as avg_duration_ms
    FROM usage_metrics um
    WHERE um.user_id = p_user_id
        AND (p_start_date IS NULL OR um.created_at >= p_start_date)
        AND (p_end_date IS NULL OR um.created_at <= p_end_date);
END;
$$;

-- Grant execute permission on the function
GRANT EXECUTE ON FUNCTION get_user_usage_summary TO authenticated;
GRANT EXECUTE ON FUNCTION get_user_usage_summary TO service_role;

COMMENT ON TABLE usage_metrics IS 'Stores detailed metrics for each AI operation including tokens, costs, and performance';
COMMENT ON COLUMN usage_metrics.operation_type IS 'Type of AI operation: data_extraction, pre_call_report, cold_email, website_scraping';
COMMENT ON COLUMN usage_metrics.input_tokens IS 'Estimated number of input tokens (based on ~4 chars per token)';
COMMENT ON COLUMN usage_metrics.output_tokens IS 'Estimated number of output tokens';
COMMENT ON COLUMN usage_metrics.estimated_cost_usd IS 'Estimated cost in USD based on model pricing';
COMMENT ON COLUMN usage_metrics.duration_ms IS 'Duration of the operation in milliseconds';
