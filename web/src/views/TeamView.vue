<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { api } from "../api";
import { formatAvgDS, formatStageDate } from "../domain/format";
import type { StageSummary, Team, TeamResult } from "../types";

const props = defineProps<{ slug: string }>();
const team = ref<Team | null>(null);
const stages = ref<StageSummary[]>([]);
const results = ref<TeamResult[]>([]);
const error = ref("");

async function load() {
  error.value = "";
  try {
    const payload = await api<{ team: Team; stages: StageSummary[] | null; results: TeamResult[] | null }>(`/api/v1/teams/${props.slug}`);
    team.value = payload.team;
    stages.value = payload.stages ?? [];
    results.value = payload.results ?? [];
  } catch (err) {
    error.value = err instanceof Error ? err.message : "команда не найдена";
  }
}

onMounted(load);
watch(() => props.slug, load);
</script>

<template>
  <p v-if="error" class="text-danger">{{ error }}</p>
  <template v-else-if="team">
    <h1 class="mb-1 text-3xl">{{ team.name }}</h1>
    <p class="mb-6 text-sm text-ink-muted">{{ team.club_name || "без клуба" }} · дом: {{ team.home_event_code }}</p>
    <div class="data-table-shell">
      <table class="data-table">
        <thead>
          <tr>
            <th>Этап</th>
            <th>Дата</th>
            <th class="cell-num">Место</th>
            <th class="cell-num">Среднее</th>
            <th class="cell-num">Очки</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(stage, index) in stages" :key="stage.id">
            <td><router-link :to="`/stages/${stage.id}`">{{ stage.event_name || stage.event_code }}</router-link></td>
            <td>{{ formatStageDate(stage.stage_date) }}</td>
            <td class="cell-num">{{ results[index]?.disqualified ? "DSQ" : results[index]?.rank ?? "—" }}</td>
            <td class="cell-num">{{ formatAvgDS(results[index]?.avg_time_ds) }}</td>
            <td class="cell-num">{{ results[index]?.total_points ?? 0 }}</td>
          </tr>
          <tr v-if="stages.length === 0">
            <td colspan="5" class="text-ink-muted">Команда ещё не выходила на этапы.</td>
          </tr>
        </tbody>
      </table>
    </div>
  </template>
</template>