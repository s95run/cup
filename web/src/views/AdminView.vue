<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { api } from "../api";
import LoginPanel from "../components/LoginPanel.vue";
import { formatStageDate } from "../domain/format";
import { useSession } from "../stores/session";
import type { Location, Season, StageDetail, StageSummary } from "../types";

const PRESET = "Единая форма команды";
const session = useSession();
const locations = ref<Location[]>([]);
const seasons = ref<Season[]>([]);
const stages = ref<StageSummary[]>([]);
const stage = ref<StageDetail | null>(null);
const seasonID = ref(0);
const stageID = ref(0);
const year = ref(new Date().getFullYear());
const title = ref("Кубок С95");
const stageNumber = ref(1);
const eventCode = ref("");
const stageDate = ref("");
const points = ref(1);
const reason = ref(PRESET);
const customReason = ref("");
const targetRegistration = ref(0);
const error = ref("");
const notice = ref("");

const reasonText = computed(() => (reason.value === "custom" ? customReason.value.trim() : reason.value));

onMounted(async () => {
  if (!session.loaded) await session.load();
  locations.value = await api<Location[]>("/api/v1/locations");
  if (locations.value[0]) eventCode.value = locations.value[0].code_name;
  if (session.user?.role === "admin") await loadAdmin();
});

watch(() => session.user?.role, async (role) => {
  if (role === "admin") await loadAdmin();
});

async function loadAdmin() {
  seasons.value = await api<Season[]>("/api/v1/seasons?all=1");
  if (!seasonID.value && seasons.value[0]) seasonID.value = seasons.value[0].id;
  await loadStages();
}

async function loadStages() {
  const season = seasons.value.find((item) => item.id === seasonID.value);
  if (!season) {
    stages.value = [];
    return;
  }
  const payload = await api<{ stages: StageSummary[] }>(`/api/v1/seasons/${season.year}`);
  stages.value = payload.stages ?? [];
  if (!stages.value.some((item) => item.id === stageID.value)) stageID.value = stages.value[0]?.id ?? 0;
  await loadStage();
}

async function loadStage() {
  stage.value = stageID.value ? await api<StageDetail>(`/api/v1/stages/${stageID.value}`) : null;
  targetRegistration.value = stage.value?.teams[0]?.registration_id ?? 0;
}

async function createSeason() {
  error.value = "";
  const created = await api<Season>("/api/v1/admin/seasons", {
    method: "POST",
    body: JSON.stringify({
      year: year.value,
      slug: `cup-${year.value}-${Date.now()}`,
      title: title.value,
      scoring_mode: "dynamic_n",
      min_stages: 2,
      published: true,
    }),
  });
  seasons.value.unshift(created);
  seasonID.value = created.id;
  notice.value = "Сезон создан";
}

async function createStage() {
  error.value = "";
  await api("/api/v1/admin/stages", {
    method: "POST",
    body: JSON.stringify({
      season_id: seasonID.value,
      stage_number: stageNumber.value,
      event_code: eventCode.value,
      stage_date: stageDate.value,
    }),
  });
  notice.value = "Этап добавлен";
  await loadStages();
}

async function syncStage() {
  error.value = "";
  stage.value = await api<StageDetail>(`/api/v1/admin/stages/${stageID.value}/sync`, { method: "POST" });
  notice.value = "Протокол синхронизирован";
}

async function addBonus() {
  error.value = "";
  if (!targetRegistration.value || reasonText.value.length < 3) {
    error.value = "Выберите команду и обоснование от 3 символов";
    return;
  }
  await api(`/api/v1/admin/registrations/${targetRegistration.value}/bonuses`, {
    method: "POST",
    body: JSON.stringify({ points: points.value, reason: reasonText.value }),
  });
  notice.value = "Баллы начислены";
  await loadStage();
}

async function removeBonus(id: number) {
  await api(`/api/v1/admin/bonuses/${id}`, { method: "DELETE" });
  notice.value = "Баллы удалены";
  await loadStage();
}
</script>

<template>
  <h1 class="mb-4 text-3xl">Админ-панель</h1>
  <LoginPanel v-if="!session.user" />
  <p v-else-if="session.user.role !== 'admin'" class="text-danger">Нужна роль администратора.</p>
  <div v-else class="space-y-6">
    <p v-if="error" class="text-danger">{{ error }}</p>
    <p v-if="notice" class="text-success">{{ notice }}</p>

    <section class="card space-y-3">
      <h2 class="text-lg">Сезон</h2>
      <form class="flex flex-wrap gap-2" @submit.prevent="createSeason">
        <input v-model.number="year" class="field w-28" type="number" required />
        <input v-model="title" class="field min-w-48 flex-1" required />
        <button class="btn btn-accent" type="submit">Создать сезон</button>
      </form>
      <select v-if="seasons.length" v-model.number="seasonID" class="field" @change="loadStages">
        <option v-for="season in seasons" :key="season.id" :value="season.id">{{ season.title }} · {{ season.year }}</option>
      </select>
    </section>

    <section class="card space-y-3">
      <h2 class="text-lg">Сетка: локация — дата</h2>
      <form class="grid gap-2 sm:grid-cols-4" @submit.prevent="createStage">
        <input v-model.number="stageNumber" class="field" type="number" min="1" required />
        <select v-model="eventCode" class="field" required>
          <option v-for="location in locations" :key="location.code_name" :value="location.code_name">{{ location.name }}</option>
        </select>
        <input v-model="stageDate" class="field" type="date" required />
        <button class="btn btn-brand" type="submit" :disabled="!seasonID">Добавить этап</button>
      </form>
      <ul class="text-sm">
        <li v-for="item in stages" :key="item.id">
          <button type="button" class="text-link" @click="stageID = item.id; loadStage()">
            {{ item.stage_number }}. {{ item.event_name || item.event_code }} — {{ formatStageDate(item.stage_date) }}
          </button>
        </li>
      </ul>
    </section>

    <section v-if="stage" class="card space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h2 class="text-lg">Этап {{ stage.stage_number }}</h2>
        <button class="btn btn-brand" type="button" @click="syncStage">Синхронизировать протокол</button>
      </div>
      <form class="grid gap-2 sm:grid-cols-4" @submit.prevent="addBonus">
        <select v-model.number="targetRegistration" class="field sm:col-span-2">
          <option v-for="team in stage.teams" :key="team.registration_id" :value="team.registration_id">{{ team.name }}</option>
        </select>
        <input v-model.number="points" class="field" type="number" min="1" max="20" required />
        <select v-model="reason" class="field">
          <option :value="PRESET">{{ PRESET }}</option>
          <option value="custom">Своё обоснование</option>
        </select>
        <input v-if="reason === 'custom'" v-model="customReason" class="field sm:col-span-3" placeholder="За что начислены баллы" />
        <button class="btn btn-accent" type="submit">Начислить</button>
      </form>
      <ul class="space-y-2 text-sm">
        <li v-for="team in stage.teams" :key="team.registration_id">
          <span class="font-medium">{{ team.name }}</span>
          <span class="text-ink-muted"> · база {{ team.base_points }} · итого {{ team.total_points }}</span>
          <span v-for="bonus in team.bonuses" :key="bonus.id" class="ml-2">
            +{{ bonus.points }} {{ bonus.reason }}
            <button type="button" class="text-danger" @click="removeBonus(bonus.id)">удалить</button>
          </span>
        </li>
        <li v-if="stage.teams.length === 0" class="text-ink-muted">На этап ещё никто не заявился.</li>
      </ul>
    </section>
  </div>
</template>