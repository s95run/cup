export const TEAM_SIZE = 5;
export const MIN_FEMALES = 2;

export interface SlotAthlete {
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

export interface RosterSummary {
  filled: number;
  females: number;
  sizeOk: boolean;
  femalesOk: boolean;
  qualifiedOk: boolean;
  duplicate: boolean;
  ready: boolean;
}

export function isQualified(runs: number, vols: number): boolean {
  return runs >= 1 || vols >= 4;
}

export function rosterSummary(athletes: Array<SlotAthlete | null>): RosterSummary {
  const picked = athletes.filter((athlete): athlete is SlotAthlete => athlete != null);
  const females = picked.filter((athlete) => athlete.gender === "female").length;
  const ids = picked.map((athlete) => athlete.id);
  const duplicate = new Set(ids).size !== ids.length;
  const qualifiedOk = picked.every((athlete) => isQualified(athlete.runs_count, athlete.volunteering_count)) && !duplicate;
  const sizeOk = picked.length === TEAM_SIZE;
  const femalesOk = females >= MIN_FEMALES;
  return {
    filled: picked.length,
    females,
    sizeOk,
    femalesOk,
    qualifiedOk,
    duplicate,
    ready: sizeOk && femalesOk && qualifiedOk,
  };
}

export interface ProtocolTeam {
  disqualified: boolean;
  rank: number | null;
}

export function splitProtocol<T extends ProtocolTeam>(teams: T[]): { classified: T[]; dsq: T[]; hasResults: boolean } {
  const dsq = teams.filter((team) => team.disqualified);
  const classified = teams.filter((team) => !team.disqualified);
  return {
    classified,
    dsq,
    hasResults: classified.some((team) => team.rank != null) || dsq.length > 0,
  };
}