import { createRouter, createWebHistory } from "vue-router";
import HomeView from "../views/HomeView.vue";
import StageView from "../views/StageView.vue";
import TeamView from "../views/TeamView.vue";
import CaptainView from "../views/CaptainView.vue";
import AdminView from "../views/AdminView.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", name: "home", component: HomeView },
    { path: "/stages/:id", name: "stage", component: StageView, props: true },
    { path: "/teams/:slug", name: "team", component: TeamView, props: true },
    { path: "/captain", name: "captain", component: CaptainView },
    { path: "/admin", name: "admin", component: AdminView },
  ],
});

export default router;