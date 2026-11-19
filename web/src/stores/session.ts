import { defineStore } from "pinia";
import { api, ApiError } from "../api";
import type { SessionUser } from "../types";

export const useSession = defineStore("session", {
  state: () => ({
    user: null as SessionUser | null,
    loaded: false,
    devLogin: false,
  }),
  actions: {
    async load() {
      try {
        this.user = await api<SessionUser>("/api/v1/auth/me");
      } catch (error) {
        if (!(error instanceof ApiError) || error.status !== 401) throw error;
        this.user = null;
      }
      try {
        await api<{ enabled: boolean }>("/api/v1/auth/dev-login");
        this.devLogin = true;
      } catch {
        this.devLogin = false;
      }
      this.loaded = true;
    },
    async loginDev(athleteID: number, name: string) {
      await api("/api/v1/auth/dev-login", {
        method: "POST",
        body: JSON.stringify({ athlete_id: athleteID, athlete_code: athleteID, name }),
      });
      this.user = await api<SessionUser>("/api/v1/auth/me");
    },
    async logout() {
      await api("/api/v1/auth/logout", { method: "POST" });
      this.user = null;
    },
  },
});