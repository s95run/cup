<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { api } from "../api";
import ProtocolTable from "../components/ProtocolTable.vue";
import { formatStageDate, registrationCountdown } from "../domain/format";
import { splitProtocol } from "../domain/roster";
import type { StageDetail } from "../types";

const props = defineProps<{ id: string }>();
const stage = ref<StageDetail | null>(null);
const error = ref("");
const now = ref(Date.now());

const split = computed(() => splitProtocol(stage.value?.teams ?? []));

async function load() {
  error.value = "";
  stage.value = null;
  try {
    stage.value = await api<StageDetail>(`/api/v1/stages/${props.id}`);
  } catch (err) {
    error.value = err instanceof Error ? err.message : "этап не найден";
  }
}

onMounted(() => {
  window.setInterval(() => {
    now.value = Date.now();
  }, 30_000);
  load();
});
watch(() => props.id, load);
</script>

<template>
  <p v-if="error" class="text-danger">{{ error }}</p>
  <template v-else-if="stage">
    <h1 class="mb-1 text-3xl">Этап {{ stage.stage_number }} · {{ stage.event_name || stage.event_code }}</h1>
    <p class="mb-6 text-sm text-ink-muted">
      {{ formatStageDate(stage.stage_date) }} · {{ registrationCountdown(stage.registration_closes_at, now) }}
    </p>
    <section v-if="split.hasResults" class="space-y-6">
      <ProtocolTable :teams="split.classified" />
      <section v-if="split.dsq.length">
        <h2 class="mb-2 text-xl">Вне зачёта</h2>
        <div class="data-table-shell">
          <table class="data-table">
            <thead>
              <tr>
                <th>Команда</th>
                <th>Причина</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="team in split.dsq" :key="team.registration_id">
                <td><router-link :to="`/teams/${team.slug}`">{{ team.name }}</router-link></td>
                <td>{{ team.dsq_reason || "неявка" }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </section>
    <section v-else>
      <h2 class="mb-2 text-xl">Заявленные команды</h2>
      <ul class="space-y-2">
        <li v-for="team in stage.teams" :key="team.registration_id" class="card">
          <router-link :to="`/teams/${team.slug}`">{{ team.name }}</router-link>
          <span class="ml-2 text-sm text-ink-muted">{{ team.members.length }} участников</span>
        </li>
        <li v-if="stage.teams.length === 0" class="text-ink-muted">Заявок нет.</li>
      </ul>
    </section>
  </template>
</template>