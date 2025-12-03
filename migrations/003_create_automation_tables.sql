-- Migration: 003_create_automation_tables
-- Description: Create automation_configs and automation_tasks tables for automated lead enrichment
-- Author: lead-gen-worker
-- Date: 2024

-- ============================================================================
-- AUTOMATION CONFIGS TABLE
-- Stores user-specific automation settings
-- ============================================================================

CREATE TABLE IF NOT EXISTS automation_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Automation toggles
    auto_enrich_new_leads BOOLEAN DEFAULT false,
    auto_generate_precall BOOLEAN DEFAULT false,
    auto_generate_email BOOLEAN DEFAULT false,
    
    -- Default business profile for automations
    default_business_profile_id UUID REFERENCES business_profiles(id) ON DELETE SET NULL,
    
    -- Limits
    daily_automation_limit INT DEFAULT 100,
    
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    
    -- Each user can only have one config
    UNIQUE(user_id)
);

-- ============================================================================
-- AUTOMATION TASKS TABLE
-- Queue for automation tasks to be processed by the worker
-- ============================================================================

CREATE TABLE IF NOT EXISTS automation_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    
    -- Task type
    task_type TEXT NOT NULL CHECK (task_type IN (
        'lead_enrichment',      -- Scrape website + Extract data
        'precall_generation',   -- Generate pre-call report
        'email_generation',     -- Generate cold email
        'full_enrichment'       -- All of the above
    )),
    
    -- Target (single lead or batch)
    lead_id UUID REFERENCES leads(id) ON DELETE CASCADE,
    lead_ids UUID[] DEFAULT '{}',
    
    -- Configuration
    business_profile_id UUID REFERENCES business_profiles(id) ON DELETE SET NULL,
    priority INT DEFAULT 2 CHECK (priority BETWEEN 1 AND 3),
    -- Priority levels:
    -- 1 = High (lead search jobs)
    -- 2 = Medium (auto-triggered enrichment)
    -- 3 = Low (manual batch operations)
    
    -- Status tracking
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    items_total INT DEFAULT 0,
    items_processed INT DEFAULT 0,
    items_succeeded INT DEFAULT 0,
    items_failed INT DEFAULT 0,
    error_message TEXT,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 2,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT now(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- Index for queue processing (pending tasks ordered by priority, then created_at)
CREATE INDEX IF NOT EXISTS idx_automation_tasks_queue 
ON automation_tasks(status, priority, created_at) 
WHERE status = 'pending';

-- Index for user's tasks history
CREATE INDEX IF NOT EXISTS idx_automation_tasks_user 
ON automation_tasks(user_id, created_at DESC);

-- Index for automation config lookup
CREATE INDEX IF NOT EXISTS idx_automation_configs_user
ON automation_configs(user_id);

-- ============================================================================
-- ROW LEVEL SECURITY (RLS)
-- ============================================================================

ALTER TABLE automation_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE automation_tasks ENABLE ROW LEVEL SECURITY;

-- Users can view their own configs
CREATE POLICY "Users can view own automation_configs"
ON automation_configs FOR SELECT
USING (auth.uid() = user_id);

-- Users can insert their own configs
CREATE POLICY "Users can insert own automation_configs"
ON automation_configs FOR INSERT
WITH CHECK (auth.uid() = user_id);

-- Users can update their own configs
CREATE POLICY "Users can update own automation_configs"
ON automation_configs FOR UPDATE
USING (auth.uid() = user_id);

-- Users can view their own tasks
CREATE POLICY "Users can view own automation_tasks"
ON automation_tasks FOR SELECT
USING (auth.uid() = user_id);

-- Users can insert their own tasks
CREATE POLICY "Users can insert own automation_tasks"
ON automation_tasks FOR INSERT
WITH CHECK (auth.uid() = user_id);

-- ============================================================================
-- SERVICE ROLE POLICIES (for worker)
-- ============================================================================

-- Service role can access all automation_configs
CREATE POLICY "Service role full access to automation_configs"
ON automation_configs FOR ALL
USING (auth.jwt()->>'role' = 'service_role');

-- Service role can access all automation_tasks
CREATE POLICY "Service role full access to automation_tasks"
ON automation_tasks FOR ALL
USING (auth.jwt()->>'role' = 'service_role');

-- ============================================================================
-- TRIGGER FOR updated_at
-- ============================================================================

CREATE OR REPLACE FUNCTION update_automation_configs_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER automation_configs_updated_at
    BEFORE UPDATE ON automation_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_automation_configs_updated_at();

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE automation_configs IS 'User-specific automation settings for lead enrichment';
COMMENT ON TABLE automation_tasks IS 'Queue of automation tasks to be processed by the worker';

COMMENT ON COLUMN automation_configs.auto_enrich_new_leads IS 'Automatically scrape and extract data when a new lead is created';
COMMENT ON COLUMN automation_configs.auto_generate_precall IS 'Automatically generate pre-call report after enrichment';
COMMENT ON COLUMN automation_configs.auto_generate_email IS 'Automatically generate cold email after pre-call report';
COMMENT ON COLUMN automation_configs.daily_automation_limit IS 'Maximum number of automated tasks per day';

COMMENT ON COLUMN automation_tasks.task_type IS 'Type of automation: lead_enrichment, precall_generation, email_generation, full_enrichment';
COMMENT ON COLUMN automation_tasks.priority IS '1=High (search jobs), 2=Medium (auto-triggered), 3=Low (manual batch)';
COMMENT ON COLUMN automation_tasks.lead_id IS 'Single lead to process';
COMMENT ON COLUMN automation_tasks.lead_ids IS 'Array of lead IDs for batch processing';
