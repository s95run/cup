<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { api } from "../api";
import ProtocolTable from "../components/ProtocolTable.vue";
import { formatStageDate, registrationCountdown } from "../domain/format";
import { splitProtocol } from "../domain/roster";
import type { Season, StageDetail, StageSummary, Standing } from "../types";

const tab = ref<"stages" | "standings">("stages");
const seasons = ref<Season[]>([]);
const year = ref<number | null>(null);
const season = ref<Season | null>(null);
const stages = ref<StageSummary[]>([]);
const details = ref<Record<number, StageDetail>>({});
const standings = ref<Standing[]>([]);
const error = ref("");
const now = ref(Date.now());

const current = computed(() => seasons.value.find((item) => item.year === year.value) ?? null);

function stageSplit(id: number) {
  return splitProtocol(details.value[id]?.teams ?? []);
}

onMounted(async () => {
  window.setInterval(() => {
    now.value = Date.now();
  }, 30_000);
  try {
    seasons.value = await api<Season[]>("/api/v1/seasons");
    if (seasons.value.length === 0) return;
    year.value = seasons.value[0].year;
    await loadYear();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "не удалось загрузить сезон";
  }
});

async function loadYear() {
  if (year.value == null) return;
  error.value = "";
  const payload = await api<{ season: Season; stages: StageSummary[] }>(`/api/v1/seasons/${year.value}`);
  season.value = payload.season;
  stages.value = payload.stages ?? [];
  standings.value = await api<Standing[]>(`/api/v1/seasons/${year.value}/standings`);
  const loaded = await Promise.all(stages.value.map((stage) => api<StageDetail>(`/api/v1/stages/${stage.id}`)));
  details.value = Object.fromEntries(loaded.map((stage) => [stage.id, stage]));
}
</script>

<template>
  <section class="mb-6">
    <h1 class="text-3xl">{{ season?.title || "Сезон Кубка С95" }}</h1>
    <p class="text-ink-muted">Этапы, заявки команд и общий зачёт.</p>
  </section>
  <div v-if="seasons.length > 1" class="mb-4">
    <select v-model.number="year" class="field max-w-xs" @change="loadYear">
      <option v-for="item in seasons" :key="item.id" :value="item.year">{{ item.year }}</option>
    </select>
  </div>
  <p v-if="error" class="mb-4 text-danger">{{ error }}</p>
  <p v-if="seasons.length === 0 && !error" class="card text-ink-muted">Сезон ещё не опубликован.</p>
  <template v-else-if="season">
    <div class="radio-tabs mb-4" role="tablist">
      <button type="button" class="radio-tab-label" :class="{ 'radio-tab-label--active': tab === 'stages' }" @click="tab = 'stages'">
        Результаты этапов
      </button>
      <button type="button" class="radio-tab-label" :class="{ 'radio-tab-label--active': tab === 'standings' }" @click="tab = 'standings'">
        Общий зачёт года
      </button>
    </div>
    <div v-if="tab === 'stages'" class="space-y-6">
      <article v-for="stage in stages" :key="stage.id" class="space-y-3">
        <header>
          <h2 class="text-xl">
            <router-link :to="`/stages/${stage.id}`">Этап {{ stage.stage_number }} · {{ stage.event_name || stage.event_code }}</router-link>
          </h2>
          <p class="text-sm text-ink-muted">
            {{ formatStageDate(stage.stage_date) }} · {{ registrationCountdown(stage.registration_closes_at, now) }}
          </p>
        </header>
        <ProtocolTable v-if="stageSplit(stage.id).hasResults" :teams="stageSplit(stage.id).classified" />
        <p v-else class="text-sm text-ink-muted">Заявлено команд: {{ details[stage.id]?.teams.length ?? 0 }}</p>
      </article>
      <p v-if="stages.length === 0" class="text-ink-muted">Сетка этапов пуста.</p>
    </div>
    <div v-else>
      <p class="mb-3 text-sm text-ink-muted">В зачёт попадают команды минимум с {{ current?.min_stages ?? season.min_stages }} этапами.</p>
      <div class="data-table-shell">
        <table class="data-table">
          <thead>
            <tr>
              <th>Место</th>
              <th>Команда</th>
              <th class="cell-num">База</th>
              <th class="cell-num">Доп.</th>
              <th class="cell-num">Итого</th>
              <th class="cell-num">Этапы</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in standings" :key="row.team_id">
              <td>{{ row.place }}</td>
              <td><router-link :to="`/teams/${row.slug}`">{{ row.name }}</router-link></td>
              <td class="cell-num">{{ row.base_points }}</td>
              <td class="cell-num">{{ row.bonus_points }}</td>
              <td class="cell-num">{{ row.points }}</td>
              <td class="cell-num">{{ row.stages }}</td>
            </tr>
            <tr v-if="standings.length === 0">
              <td colspan="6" class="text-ink-muted">Пока никто не набрал нужное число этапов.</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </template>
</template>