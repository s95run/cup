<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useSession } from "./stores/session";

const session = useSession();
const dark = ref(document.documentElement.classList.contains("dark"));

onMounted(() => {
  session.load().catch(() => {
    session.loaded = true;
  });
});

function toggleTheme() {
  dark.value = !dark.value;
  document.documentElement.classList.toggle("dark", dark.value);
  localStorage.setItem("cup-theme", dark.value ? "dark" : "light");
}

async function logout() {
  await session.logout();
}
</script>

<template>
  <div class="min-h-screen bg-canvas text-ink">
    <header class="border-b border-white/10 bg-nav text-nav-fg">
      <div class="page-container flex h-14 items-center justify-between gap-3">
        <router-link to="/" class="font-display text-lg text-nav-fg no-underline">Кубок С95</router-link>
        <nav class="flex items-center gap-3 text-sm">
          <router-link to="/captain" class="text-nav-fg no-underline">Капитан</router-link>
          <router-link to="/admin" class="text-nav-fg no-underline">Админ</router-link>
          <button type="button" class="text-nav-fg" @click="toggleTheme">{{ dark ? "Светлая" : "Тёмная" }}</button>
          <button v-if="session.user" type="button" class="text-nav-fg" @click="logout">Выйти</button>
          <a v-else href="/api/v1/auth/s95/login" class="text-nav-fg">Войти</a>
        </nav>
      </div>
    </header>
    <main class="page-container py-8">
      <router-view />
    </main>
  </div>
</template>