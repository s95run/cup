-- +goose Up
CREATE TABLE IF NOT EXISTS locations_cache (
    code_name TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    town TEXT NOT NULL DEFAULT '',
    timezone TEXT NOT NULL DEFAULT 'Europe/Moscow',
    active BOOLEAN NOT NULL DEFAULT true,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    s95_user_id BIGINT UNIQUE,
    s95_athlete_id BIGINT UNIQUE NOT NULL,
    athlete_code BIGINT NOT NULL,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'captain' CHECK (role IN ('captain', 'admin')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS used_sso_tokens (
    jti TEXT PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS seasons (
    id BIGSERIAL PRIMARY KEY,
    year INT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    scoring_mode TEXT NOT NULL DEFAULT 'dynamic_n' CHECK (scoring_mode IN ('dynamic_n', 'avg_time')),
    min_stages_required INT NOT NULL DEFAULT 2,
    scoring_config JSONB NOT NULL DEFAULT '{"bonus_presets":["Единая форма команды"]}'::jsonb,
    is_published BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS teams (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    home_event_code TEXT NOT NULL REFERENCES locations_cache (code_name),
    club_name TEXT NOT NULL DEFAULT '',
    captain_user_id BIGINT NOT NULL REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS stages (
    id BIGSERIAL PRIMARY KEY,
    season_id BIGINT NOT NULL REFERENCES seasons (id) ON DELETE CASCADE,
    stage_number INT NOT NULL,
    event_code TEXT NOT NULL REFERENCES locations_cache (code_name),
    stage_date DATE NOT NULL,
    registration_closes_at TIMESTAMPTZ NOT NULL,
    s95_activity_id BIGINT,
    s95_activity_updated_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'upcoming' CHECK (status IN ('upcoming', 'registration_closed', 'completed')),
    UNIQUE (season_id, stage_number)
);

CREATE TABLE IF NOT EXISTS stage_registrations (
    id BIGSERIAL PRIMARY KEY,
    stage_id BIGINT NOT NULL REFERENCES stages (id) ON DELETE CASCADE,
    team_id BIGINT NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (stage_id, team_id)
);

CREATE TABLE IF NOT EXISTS stage_roster_members (
    id BIGSERIAL PRIMARY KEY,
    registration_id BIGINT NOT NULL REFERENCES stage_registrations (id) ON DELETE CASCADE,
    stage_id BIGINT NOT NULL REFERENCES stages (id) ON DELETE CASCADE,
    s95_athlete_id BIGINT NOT NULL,
    athlete_name TEXT NOT NULL,
    gender TEXT NOT NULL CHECK (gender IN ('male', 'female')),
    home_event_snapshot TEXT NOT NULL DEFAULT '',
    club_snapshot TEXT NOT NULL DEFAULT '',
    qualification_runs INT NOT NULL DEFAULT 0,
    qualification_vols INT NOT NULL DEFAULT 0,
    finish_time_sec INT,
    activity_position INT,
    is_dnf BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (stage_id, s95_athlete_id)
);

CREATE TABLE IF NOT EXISTS stage_bonus_awards (
    id BIGSERIAL PRIMARY KEY,
    registration_id BIGINT NOT NULL REFERENCES stage_registrations (id) ON DELETE CASCADE,
    stage_id BIGINT NOT NULL REFERENCES stages (id) ON DELETE CASCADE,
    points INT NOT NULL CHECK (points > 0),
    reason TEXT NOT NULL,
    awarded_by_user_id BIGINT NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS stage_team_results (
    registration_id BIGINT PRIMARY KEY REFERENCES stage_registrations (id) ON DELETE CASCADE,
    stage_id BIGINT NOT NULL REFERENCES stages (id) ON DELETE CASCADE,
    is_disqualified BOOLEAN NOT NULL DEFAULT false,
    dsq_reason TEXT NOT NULL DEFAULT '',
    total_time_sec INT,
    avg_time_ds INT,
    fifth_runner_time_sec INT,
    rank INT,
    base_points INT NOT NULL DEFAULT 0,
    bonus_points INT NOT NULL DEFAULT 0,
    total_points INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_stages_season ON stages (season_id);
CREATE INDEX IF NOT EXISTS idx_stage_registrations_stage ON stage_registrations (stage_id);
CREATE INDEX IF NOT EXISTS idx_used_sso_tokens_expires ON used_sso_tokens (expires_at);

-- +goose Down
DROP TABLE IF EXISTS stage_team_results;
DROP TABLE IF EXISTS stage_bonus_awards;
DROP TABLE IF EXISTS stage_roster_members;
DROP TABLE IF EXISTS stage_registrations;
DROP TABLE IF EXISTS stages;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS seasons;
DROP TABLE IF EXISTS used_sso_tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS locations_cache;