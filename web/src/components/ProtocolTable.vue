<script setup lang="ts">
import { ref } from "vue";
import { formatAvgDS, formatFinish } from "../domain/format";
import type { TeamResult } from "../types";

defineProps<{ teams: TeamResult[] }>();
const open = ref<Record<number, boolean>>({});

function toggle(id: number) {
  open.value[id] = !open.value[id];
}

function bonusTitle(team: TeamResult): string {
  return team.bonuses.map((bonus) => `+${bonus.points} — ${bonus.reason}`).join("\n");
}

function athleteURL(id: number): string {
  return `https://s95.ru/athletes/${id}`;
}
</script>

<template>
  <div class="data-table-shell">
    <table class="data-table">
      <thead>
        <tr>
          <th>Место</th>
          <th>Команда</th>
          <th class="cell-num">Среднее</th>
          <th class="cell-num">База</th>
          <th class="cell-num">Доп.</th>
          <th class="cell-num">Итого</th>
        </tr>
      </thead>
      <tbody>
        <template v-for="team in teams" :key="team.registration_id">
          <tr class="row-button" @click="toggle(team.registration_id)">
            <td>{{ team.rank ?? "—" }}</td>
            <td>
              <router-link :to="`/teams/${team.slug}`" @click.stop>{{ team.name }}</router-link>
              <span v-if="team.bonus_points > 0" class="badge-bonus ml-2" :title="bonusTitle(team)">
                +{{ team.bonus_points }} доп.
              </span>
            </td>
            <td class="cell-num">{{ formatAvgDS(team.avg_time_ds) }}</td>
            <td class="cell-num">{{ team.base_points }}</td>
            <td class="cell-num">{{ team.bonus_points }}</td>
            <td class="cell-num">{{ team.total_points }}</td>
          </tr>
          <tr v-if="open[team.registration_id]">
            <td colspan="6" class="bg-surface-elevated">
              <ul class="space-y-1">
                <li v-for="member in team.members" :key="member.athlete_id" class="flex flex-wrap gap-x-3 text-sm">
                  <a :href="athleteURL(member.athlete_id)" target="_blank" rel="noreferrer">{{ member.name }}</a>
                  <span>{{ formatFinish(member.finish_time_sec) }}</span>
                  <span v-if="member.position">место {{ member.position }}</span>
                  <span v-if="member.club" class="text-ink-muted">{{ member.club }}</span>
                  <span v-if="member.home_event" class="text-ink-muted">{{ member.home_event }}</span>
                  <span v-if="member.dnf" class="text-danger">не финишировал</span>
                </li>
              </ul>
              <p v-if="team.bonuses.length" class="mt-2 text-sm text-ink-muted">
                <span v-for="bonus in team.bonuses" :key="bonus.id" class="mr-3">+{{ bonus.points }} — {{ bonus.reason }}</span>
              </p>
            </td>
          </tr>
        </template>
        <tr v-if="teams.length === 0">
          <td colspan="6" class="text-ink-muted">Команд пока нет</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>