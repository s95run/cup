<script setup lang="ts">
import { ref } from "vue";
import { useSession } from "../stores/session";

const session = useSession();
const athleteID = ref(99);
const name = ref("Капитан");
const error = ref("");
const pending = ref(false);

async function submit() {
  error.value = "";
  pending.value = true;
  try {
    await session.loginDev(athleteID.value, name.value);
  } catch (err) {
    error.value = err instanceof Error ? err.message : "не удалось войти";
  } finally {
    pending.value = false;
  }
}
</script>

<template>
  <section class="card space-y-3">
    <h2 class="text-lg">Вход</h2>
    <a class="btn btn-brand" href="/api/v1/auth/s95/login">Войти через S95</a>
    <form v-if="session.devLogin" class="space-y-2" @submit.prevent="submit">
      <p class="text-sm text-ink-muted">
        Локальный вход. Админ — по <code>admin_athlete_ids</code> (внутренний <code>athlete_id</code> в S95, например 1), капитан — 99.
      </p>
      <label class="block text-sm">
        ID атлета (S95)
        <input v-model.number="athleteID" class="field mt-1" type="number" required />
      </label>
      <label class="block text-sm">
        Имя
        <input v-model="name" class="field mt-1" required />
      </label>
      <button class="btn btn-accent" type="submit" :disabled="pending">Войти локально</button>
      <p v-if="error" class="text-sm text-danger">{{ error }}</p>
    </form>
  </section>
</template>