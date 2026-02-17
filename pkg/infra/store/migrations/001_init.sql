-- AIMA Database Schema v1
-- Initial migration creating core tables

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Models: AI models managed by the system
CREATE TABLE IF NOT EXISTS models (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,           -- llm, embedding, image, audio, etc.
    format TEXT,                  -- gguf, safetensors, onnx, etc.
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, downloading, ready, error
    source TEXT,                  -- ollama, huggingface, modelscope, local
    path TEXT,                    -- filesystem path to model files
    size INTEGER DEFAULT 0,       -- size in bytes
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Engines: Inference engines (ollama, vllm, tensorrt-llm, etc.)
CREATE TABLE IF NOT EXISTS engines (
    name TEXT PRIMARY KEY,
    type TEXT NOT NULL,           -- ollama, vllm, tensorrt-llm, etc.
    status TEXT NOT NULL DEFAULT 'stopped',  -- stopped, running, error
    version TEXT,
    config TEXT,                  -- JSON configuration
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Services: Running inference services
CREATE TABLE IF NOT EXISTS services (
    id TEXT PRIMARY KEY,
    model_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, running, stopped, error
    replicas INTEGER DEFAULT 1,
    config TEXT,                  -- JSON configuration
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (model_id) REFERENCES models(id)
);

-- Apps: Application deployments
CREATE TABLE IF NOT EXISTS apps (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    template TEXT,                -- template name or ID
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, running, stopped, error
    ports TEXT,                   -- JSON array of port mappings
    volumes TEXT,                 -- JSON array of volume mounts
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Pipelines: Processing pipelines
CREATE TABLE IF NOT EXISTS pipelines (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    steps TEXT,                   -- JSON array of pipeline steps
    config TEXT,                  -- JSON configuration
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, running, paused, completed, error
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Alerts: System alerts and notifications
CREATE TABLE IF NOT EXISTS alerts (
    id TEXT PRIMARY KEY,
    rule_id TEXT NOT NULL,        -- reference to alert rule
    status TEXT NOT NULL DEFAULT 'active',   -- active, acknowledged, resolved
    severity TEXT NOT NULL DEFAULT 'info',   -- info, warning, error, critical
    message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Allocations: Resource allocations (GPU slots, memory, etc.)
CREATE TABLE IF NOT EXISTS allocations (
    slot_id TEXT PRIMARY KEY,     -- GPU slot identifier (e.g., "gpu:0", "gpu:1")
    type TEXT NOT NULL,           -- gpu, cpu, memory
    memory INTEGER DEFAULT 0,     -- allocated memory in MB
    status TEXT NOT NULL DEFAULT 'free',     -- free, allocated, reserved
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_models_status ON models(status);
CREATE INDEX IF NOT EXISTS idx_models_name ON models(name);
CREATE INDEX IF NOT EXISTS idx_services_model_id ON services(model_id);
CREATE INDEX IF NOT EXISTS idx_services_status ON services(status);
CREATE INDEX IF NOT EXISTS idx_engines_status ON engines(status);
CREATE INDEX IF NOT EXISTS idx_apps_status ON apps(status);
CREATE INDEX IF NOT EXISTS idx_pipelines_status ON pipelines(status);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
CREATE INDEX IF NOT EXISTS idx_allocations_status ON allocations(status);

-- Record initial schema version
INSERT OR IGNORE INTO schema_version (version) VALUES (1);
