import { createRouter, createWebHashHistory } from "vue-router";
import AppDetailView from "../views/AppDetailView.vue";
import MarketView from "../views/MarketView.vue";
import MineView from "../views/MineView.vue";
import SettingsAboutView from "../views/SettingsAboutView.vue";
import SettingsDownloadView from "../views/SettingsDownloadView.vue";
import SettingsModalView from "../views/SettingsModalView.vue";

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: "/",
      redirect: "/market",
    },
    {
      path: "/market",
      name: "market",
      component: MarketView,
    },
    {
      path: "/market/app/:category/:name",
      name: "app-detail",
      component: AppDetailView,
    },
    {
      path: "/mine",
      name: "mine",
      component: MineView,
    },
    {
      path: "/settings",
      component: SettingsModalView,
      children: [
        {
          path: "",
          redirect: "/settings/download",
        },
        {
          path: "download",
          name: "settings-download",
          component: SettingsDownloadView,
        },
        {
          path: "about",
          name: "settings-about",
          component: SettingsAboutView,
        },
      ],
    },
  ],
});

export default router;
