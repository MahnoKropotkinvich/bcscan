-- ============================================================
-- 007: 增强告警管理 + 审计日志 (Batch 5)
-- ============================================================

-- 给 alerts 表添加 acknowledged_by 字段（谁确认的）
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS acknowledged_by BIGINT REFERENCES users(id) ON DELETE SET NULL;

-- 给 alerts 表添加 resolved_by 字段（谁处理的）
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS resolved_by BIGINT REFERENCES users(id) ON DELETE SET NULL;

-- 给 alerts 表添加 updated_at 字段
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- 告警处理历史表（记录告警状态变更全过程）
CREATE TABLE IF NOT EXISTS alert_history (
    id BIGSERIAL PRIMARY KEY,
    alert_id BIGINT NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,        -- created, acknowledged, resolved, ignored, reopened, note_added
    old_status VARCHAR(20),
    new_status VARCHAR(20),
    note TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_alert_history_alert ON alert_history(alert_id);
CREATE INDEX IF NOT EXISTS idx_alert_history_user ON alert_history(user_id);
CREATE INDEX IF NOT EXISTS idx_alert_history_created ON alert_history(created_at);

-- 自动创建告警的触发器函数：risk_events 插入后自动在 alerts 表创建对应告警
CREATE OR REPLACE FUNCTION create_alert_from_risk_event()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO alerts (risk_event_id, title, message, severity, status, created_at)
    VALUES (
        NEW.id,
        NEW.event_type,
        NEW.description,
        NEW.severity,
        'pending',
        NOW()
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 触发器（仅在不存在时创建）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'trg_create_alert_on_risk_event'
    ) THEN
        CREATE TRIGGER trg_create_alert_on_risk_event
        AFTER INSERT ON risk_events
        FOR EACH ROW
        EXECUTE FUNCTION create_alert_from_risk_event();
    END IF;
END $$;

-- 确保 audit_logs 表有完整的字段
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS request_method VARCHAR(10);
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS request_path VARCHAR(500);
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS status_code INTEGER;
