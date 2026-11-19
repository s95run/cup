<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { api } from "../api";
import LoginPanel from "../components/LoginPanel.vue";
import { formatStageDate } from "../domain/format";
import { isQualified, rosterSummary, type SlotAthlete } from "../domain/roster";
import { useSession } from "../stores/session";
import type { AthleteHit, Location, Season, StageSummary, Team } from "../types";

interface Slot {
  athlete: SlotAthlete | null;
  query: string;
  hits: AthleteHit[];
}

const session = useSession();
const locations = ref<Location[]>([]);
const teams = ref<Team[]>([]);
const stages = ref<StageSummary[]>([]);
const teamID = ref(0);
const stageID = ref(0);
const teamName = ref("");
const homeEvent = ref("");
const clubName = ref("");
const slots = ref<Slot[]>(emptySlots());
const error = ref("");
const notice = ref("");
const registrationID = ref<number | null>(null);
let searchTimer = 0;

const summary = computed(() => rosterSummary(slots.value.map((slot) => slot.athlete)));
const openStages = computed(() => stages.value.filter((stage) => new Date(stage.registration_closes_at).getTime() > Date.now()));

function emptySlots(): Slot[] {
  return Array.from({ length: 5 }, () => ({ athlete: null, query: "", hits: [] }));
}

function draftKey() {
  return `cup_roster_draft_${teamID.value}_${stageID.value}`;
}

function saveDraft() {
  if (!teamID.value || !stageID.value) return;
  const athletes = slots.value.map((slot) => slot.athlete).filter((athlete): athlete is SlotAthlete => athlete != null);
  if (athletes.length === 0) {
    localStorage.removeItem(draftKey());
    return;
  }
  localStorage.setItem(draftKey(), JSON.stringify(athletes));
}

function loadDraft(): boolean {
  const raw = localStorage.getItem(draftKey());
  if (!raw) return false;
  try {
    const athletes = JSON.parse(raw) as SlotAthlete[];
    slots.value = emptySlots();
    athletes.slice(0, 5).forEach((athlete, index) => {
      slots.value[index].athlete = athlete;
      slots.value[index].query = athlete.name;
    });
    return athletes.length > 0;
  } catch {
    localStorage.removeItem(draftKey());
    return false;
  }
}

function toSlot(hit: AthleteHit): SlotAthlete {
  return { ...hit };
}

onMounted(async () => {
  if (!session.loaded) await session.load();
  locations.value = await api<Location[]>("/api/v1/locations");
  if (locations.value.length && !homeEvent.value) homeEvent.value = locations.value[0].code_name;
  if (session.user) await loadCabinet();
});

watch(() => session.user?.user_id, async (id) => {
  if (id) await loadCabinet();
});

watch(
  () => slots.value.map((slot) => slot.athlete?.id),
  () => saveDraft(),
  { deep: true },
);

async function loadCabinet() {
  error.value = "";
  teams.value = await api<Team[]>("/api/v1/captain/teams");
  const seasons = await api<Season[]>("/api/v1/seasons");
  const lists = await Promise.all(seasons.map((season) => api<{ stages: StageSummary[] }>(`/api/v1/seasons/${season.year}`)));
  stages.value = lists.flatMap((list) => list.stages ?? []);
  if (!teamID.value && teams.value.length) teamID.value = teams.value[0].id;
  if (!stageID.value && openStages.value.length) stageID.value = openStages.value[0].id;
  await loadExistingRoster();
}

async function loadExistingRoster() {
  registrationID.value = null;
  slots.value = emptySlots();
  if (!stageID.value || !teamID.value) return;
  if (loadDraft()) {
    try {
      const saved = await api<{ registration_id: number; athletes: AthleteHit[] }>(
        `/api/v1/stages/${stageID.value}/registration?team_id=${teamID.value}`,
      );
      registrationID.value = saved.registration_id;
    } catch {
      /* draft only */
    }
    return;
  }
  try {
    const saved = await api<{ registration_id: number; athletes: AthleteHit[] }>(
      `/api/v1/stages/${stageID.value}/registration?team_id=${teamID.value}`,
    );
    registrationID.value = saved.registration_id;
    saved.athletes.slice(0, 5).forEach((hit, index) => {
      slots.value[index].athlete = toSlot(hit);
      slots.value[index].query = hit.name;
    });
  } catch {
    registrationID.value = null;
  }
}

async function createTeam() {
  error.value = "";
  notice.value = "";
  try {
    const team = await api<Team>("/api/v1/teams", {
      method: "POST",
      body: JSON.stringify({ name: teamName.value, home_event_code: homeEvent.value, club_name: clubName.value }),
    });
    teams.value.push(team);
    teamID.value = team.id;
    teamName.value = "";
    clubName.value = "";
    notice.value = "Команда создана";
  } catch (err) {
    error.value = err instanceof Error ? err.message : "не удалось создать команду";
  }
}

function scheduleSearch(index: number) {
  window.clearTimeout(searchTimer);
  searchTimer = window.setTimeout(() => search(index), 250);
}

async function search(index: number) {
  const slot = slots.value[index];
  if (slot.query.trim().length < 2) {
    slot.hits = [];
    return;
  }
  const payload = await api<{ athletes: AthleteHit[] }>(`/api/v1/athletes/search?q=${encodeURIComponent(slot.query.trim())}`);
  slot.hits = payload.athletes ?? [];
}

function pick(index: number, hit: AthleteHit) {
  slots.value[index].athlete = toSlot(hit);
  slots.value[index].query = hit.name;
  slots.value[index].hits = [];
  saveDraft();
}

function clearSlot(index: number) {
  slots.value[index] = { athlete: null, query: "", hits: [] };
  saveDraft();
}

async function saveRoster() {
  error.value = "";
  notice.value = "";
  const athletes = slots.value.map((slot) => slot.athlete).filter((athlete): athlete is SlotAthlete => athlete != null);
  try {
    const saved = await api<{ registration_id: number }>(`/api/v1/stages/${stageID.value}/registration`, {
      method: "PUT",
      body: JSON.stringify({
        team_id: teamID.value,
        athletes,
      }),
    });
    registrationID.value = saved.registration_id;
    notice.value = "Заявка сохранена";
    localStorage.removeItem(draftKey());
  } catch (err) {
    error.value = err instanceof Error ? err.message : "заявка не сохранена";
  }
}

async function withdraw() {
  error.value = "";
  await api(`/api/v1/stages/${stageID.value}/registration?team_id=${teamID.value}`, { method: "DELETE" });
  registrationID.value = null;
  slots.value = emptySlots();
  localStorage.removeItem(draftKey());
  notice.value = "Заявка снята";
}
</script>

<template>
  <h1 class="mb-4 text-3xl">Кабинет капитана</h1>
  <LoginPanel v-if="!session.user" />
  <div v-else class="space-y-6">
    <p v-if="error" class="text-danger">{{ error }}</p>
    <p v-if="notice" class="text-success">{{ notice }}</p>

    <section class="card space-y-3">
      <h2 class="text-lg">Новая команда</h2>
      <form class="grid gap-3 sm:grid-cols-3" @submit.prevent="createTeam">
        <input v-model="teamName" class="field" placeholder="Название" required />
        <select v-model="homeEvent" class="field" required>
          <option v-for="location in locations" :key="location.code_name" :value="location.code_name">
            {{ location.name }}
          </option>
        </select>
        <input v-model="clubName" class="field" placeholder="Клуб" />
        <button class="btn btn-accent sm:col-span-3 sm:w-fit" type="submit">Создать</button>
      </form>
    </section>

    <section v-if="teams.length === 0" class="text-ink-muted">Сначала создайте команду.</section>
    <section v-else class="space-y-4">
      <div class="grid gap-3 sm:grid-cols-2">
        <label class="text-sm">
          Команда
          <select v-model.number="teamID" class="field mt-1" @change="loadExistingRoster">
            <option v-for="team in teams" :key="team.id" :value="team.id">{{ team.name }}</option>
          </select>
        </label>
        <label class="text-sm">
          Этап
          <select v-model.number="stageID" class="field mt-1" @change="loadExistingRoster">
            <option v-for="stage in openStages" :key="stage.id" :value="stage.id">
              {{ stage.event_name || stage.event_code }} · {{ formatStageDate(stage.stage_date) }}
            </option>
          </select>
        </label>
      </div>
      <p v-if="openStages.length === 0" class="text-ink-muted">Нет этапов с открытой регистрацией.</p>

      <div class="flex flex-wrap gap-2 text-sm">
        <span :class="summary.sizeOk ? 'badge-ok' : 'badge-bad'">{{ summary.filled }}/5 чел.</span>
        <span :class="summary.femalesOk ? 'badge-ok' : 'badge-bad'">девушек {{ summary.females }} / минимум 2</span>
        <span :class="summary.qualifiedOk ? 'badge-ok' : 'badge-bad'">
          {{ summary.duplicate ? "атлет повторяется" : "квалификация" }}
        </span>
      </div>

      <ol class="space-y-3">
        <li v-for="(slot, index) in slots" :key="index" class="card">
          <div class="mb-2 flex items-center justify-between">
            <span class="text-ink-muted">Слот {{ index + 1 }}</span>
            <button v-if="slot.athlete || slot.query" type="button" class="text-sm text-link" @click="clearSlot(index)">Очистить</button>
          </div>
          <input v-model="slot.query" class="field" placeholder="Имя участника" @input="scheduleSearch(index)" />
          <ul v-if="slot.hits.length" class="mt-2 divide-y divide-line rounded-md border border-line">
            <li v-for="hit in slot.hits" :key="hit.id">
              <button type="button" class="w-full px-3 py-2 text-left text-sm hover:bg-surface-elevated" @click="pick(index, hit)">
                {{ hit.name }}
                <span class="text-ink-muted">
                  · {{ hit.gender === "female" ? "ж" : "м" }} · забеги {{ hit.runs_count }} · волонтёрства {{ hit.volunteering_count }}
                </span>
              </button>
            </li>
          </ul>
          <p v-if="slot.athlete" class="mt-2 text-sm">
            <span :class="isQualified(slot.athlete.runs_count, slot.athlete.volunteering_count) ? 'badge-ok' : 'badge-bad'">
              {{ isQualified(slot.athlete.runs_count, slot.athlete.volunteering_count) ? "допущен" : "нет квалификации" }}
            </span>
            <span class="ml-2 text-ink-muted">{{ slot.athlete.gender === "female" ? "девушка" : "мужчина" }}</span>
          </p>
        </li>
      </ol>
      <div class="flex gap-2">
        <button class="btn btn-accent" type="button" :disabled="!summary.ready || !stageID" @click="saveRoster">Сохранить заявку</button>
        <button v-if="registrationID" class="btn btn-ghost" type="button" @click="withdraw">Снять заявку</button>
      </div>
    </section>
  </div>
</template>