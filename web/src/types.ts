export interface SessionUser {
  user_id: number;
  athlete_id: number;
  role: string;
  name: string;
}

export interface Location {
  code_name: string;
  name: string;
  town: string;
  timezone: string;
  active: boolean;
}

export interface Season {
  id: number;
  year: number;
  slug: string;
  title: string;
  scoring_mode: string;
  min_stages: number;
  published: boolean;
}

export interface StageSummary {
  id: number;
  season_id: number;
  stage_number: number;
  event_code: string;
  event_name: string;
  timezone: string;
  stage_date: string;
  registration_closes_at: string;
  status: string;
}

export interface Standing {
  place: number;
  team_id: number;
  name: string;
  slug: string;
  points: number;
  base_points: number;
  bonus_points: number;
  stages: number;
  first_places: number;
}

export interface Member {
  athlete_id: number;
  name: string;
  gender: string;
  home_event: string;
  club: string;
  finish_time_sec: number | null;
  position: number | null;
  dnf: boolean;
}

export interface Bonus {
  id: number;
  points: number;
  reason: string;
}

export interface TeamResult {
  registration_id: number;
  team_id: number;
  name: string;
  slug: string;
  home_event_code: string;
  club_name: string;
  rank: number | null;
  total_time_sec: number | null;
  avg_time_ds: number | null;
  base_points: number;
  bonus_points: number;
  total_points: number;
  disqualified: boolean;
  dsq_reason: string;
  bonuses: Bonus[];
  members: Member[];
}

export interface StageDetail {
  id: number;
  season_id: number;
  stage_number: number;
  event_code: string;
  event_name: string;
  stage_date: string;
  registration_closes_at: string;
  status: string;
  teams: TeamResult[];
}

export interface AthleteHit {
  id: number;
  code: number;
  name: string;
  gender: string;
  home_event_code: string;
  home_event_name: string;
  club_name: string;
  runs_count: number;
  volunteering_count: number;
  exp: number;
  sig: string;
}

export interface Team {
  id: number;
  name: string;
  slug: string;
  home_event_code: string;
  club_name: string;
  captain_user_id: number;
}