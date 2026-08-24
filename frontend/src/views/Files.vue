<template>
  <div>
    <header-bar
      v-if="error || fileStore.req?.type === undefined"
      showMenu
      showLogo
    />

    <breadcrumbs base="/files" />
    <errors v-if="error" :errorCode="error.status" />
    <component v-else-if="currentView" :is="currentView"></component>
    <div v-else>
      <h2 class="message delayed">
        <div class="spinner">
          <div class="bounce1"></div>
          <div class="bounce2"></div>
          <div class="bounce3"></div>
        </div>
        <span>{{ t("files.loading") }}</span>
      </h2>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  computed,
  defineAsyncComponent,
  onBeforeUnmount,
  onMounted,
  onUnmounted,
  nextTick,
  ref,
  watch,
} from "vue";
import { files as api } from "@/api";
import { storeToRefs } from "pinia";
import { useFileStore } from "@/stores/file";
import { useLayoutStore } from "@/stores/layout";

import HeaderBar from "@/components/header/HeaderBar.vue";
import Breadcrumbs from "@/components/Breadcrumbs.vue";
import Errors from "@/views/Errors.vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import FileListing from "@/views/files/FileListing.vue";
import { StatusError } from "@/api/utils";
import { name } from "../utils/constants";

const Editor = defineAsyncComponent(() => import("@/views/files/Editor.vue"));
const Preview = defineAsyncComponent(() => import("@/views/files/Preview.vue"));

const layoutStore = useLayoutStore();
const fileStore = useFileStore();

const { reload } = storeToRefs(fileStore);

const route = useRoute();

const { t } = useI18n({});

let fetchDataController = new AbortController();

const error = ref<StatusError | null>(null);

const currentView = computed(() => {
  if (fileStore.req?.type === undefined) {
    return null;
  }

  if (fileStore.req.isDir) {
    return FileListing;
  } else if (fileStore.req.extension.toLowerCase() === ".csv") {
    // CSV files use Preview for table view, unless ?edit=true
    if (route.query.edit === "true") {
      return Editor;
    }
    return Preview;
  } else if (
    fileStore.req.type === "text" ||
    fileStore.req.type === "textImmutable"
  ) {
    return Editor;
  } else {
    return Preview;
  }
});

// --- Автообновление списка + индикатор-«часы» (кастом) ---
// Интервал живёт в fileStore.autoRefreshIntervalMs — единственный источник правды:
// им же (через инлайн animation-duration) управляется CSS-анимация кольца в FileListing.
// Модель: setTimeout, который перезапланируется после каждого обновления/навигации.
// В начале каждого отсчёта дёргаем fileStore.refreshCycle → кольцо (через :key)
// перезапускает анимацию, поэтому «12 часов» кольца совпадает с моментом обновления.
// silentRefresh, в отличие от fetchData(), не трогает loading/спиннер и не сбрасывает
// выделение: updateRequest() сам восстанавливает selection по url элементов.
let autoRefreshTimer: number | undefined;
let silentController = new AbortController();

const currentUrl = () => {
  let url = route.path;
  if (url === "") url = "/";
  if (url[0] !== "/") url = "/" + url;
  return url;
};

const silentRefresh = async () => {
  // только для листинга папки, активной вкладки, без открытых модалок/меню
  if (
    !fileStore.req?.isDir ||
    document.hidden ||
    layoutStore.loading ||
    layoutStore.prompts.length > 0 ||
    fileStore.contextMenuOpen
  ) {
    return;
  }

  const url = currentUrl();
  silentController = new AbortController();
  try {
    const res = await api.fetch(url, silentController.signal);
    // отбрасываем ответ, если за время запроса ушли в другую папку
    if (currentUrl() === url) {
      // флаг живёт ровно на время реакции на смену req: листинг по нему
      // не сбрасывает showLimit и не скроллит к выделенному элементу
      fileStore.isSilentRefresh = true;
      fileStore.updateRequest(res);
      await nextTick();
      fileStore.isSilentRefresh = false;
    }
  } catch {
    // тихо игнорируем — ошибки обработает обычный fetchData при навигации
  }
};

const runAutoRefresh = async () => {
  await silentRefresh();
  scheduleAutoRefresh();
};

const scheduleAutoRefresh = () => {
  if (autoRefreshTimer) window.clearTimeout(autoRefreshTimer);
  fileStore.refreshCycle++; // рестарт анимации кольца (через :key в FileListing)
  autoRefreshTimer = window.setTimeout(
    runAutoRefresh,
    fileStore.autoRefreshIntervalMs
  );
};

// Define hooks
onMounted(() => {
  fetchData();
  fileStore.isFiles = true;
  window.addEventListener("keydown", keyEvent);
});

onBeforeUnmount(() => {
  window.removeEventListener("keydown", keyEvent);
});

onUnmounted(() => {
  fileStore.isFiles = false;
  if (layoutStore.showShell) {
    layoutStore.toggleShell();
  }
  fileStore.updateRequest(null);
  fetchDataController.abort();
  if (autoRefreshTimer) window.clearTimeout(autoRefreshTimer);
  silentController.abort();
});

watch(route, () => {
  fetchData();
});
watch(reload, (newValue) => {
  newValue && fetchData();
});
// ручное обновление по кнопке-кольцу: тот же тихий путь, что и по таймеру,
// плюс перезапуск отсчёта (scheduleAutoRefresh внутри runAutoRefresh)
watch(
  () => fileStore.refreshRequest,
  () => runAutoRefresh()
);

// Define functions

const applyPreSelection = () => {
  const preselect = fileStore.preselect;
  fileStore.preselect = null;

  if (!fileStore.req?.isDir || fileStore.oldReq === null) return;

  let index = -1;
  if (preselect) {
    // Find item with the specified path
    index = fileStore.req.items.findIndex((item) => item.path === preselect);
  } else if (fileStore.oldReq.path.startsWith(fileStore.req.path)) {
    // Get immediate child folder of the previous path
    const name = fileStore.oldReq.path
      .substring(fileStore.req.path.length)
      .split("/")
      .shift();

    index = fileStore.req.items.findIndex(
      (val) => val.path == fileStore.req!.path + name
    );
  }

  if (index === -1) return;
  fileStore.selected.push(index);
};

const fetchData = async () => {
  // Reset view information.
  fileStore.reload = false;
  fileStore.selected = [];
  fileStore.multiple = false;
  layoutStore.closeHovers();

  // Set loading to true and reset the error.
  layoutStore.loading = true;
  error.value = null;

  let url = route.path;
  if (url === "") url = "/";
  if (url[0] !== "/") url = "/" + url;
  // Cancel the ongoing request
  fetchDataController.abort();
  fetchDataController = new AbortController();
  try {
    const res = await api.fetch(url, fetchDataController.signal);
    fileStore.updateRequest(res);
    document.title = `${res.name || t("sidebar.myFiles")} - ${t("files.files")} - ${name}`;
    layoutStore.loading = false;

    // Selects the post-reload target item or the previously visited child folder
    applyPreSelection();

    // перезапустить отсчёт до следующего авто-обновления (и синхронизировать кольцо)
    scheduleAutoRefresh();
  } catch (err) {
    if (err instanceof StatusError && err.is_canceled) {
      return;
    }
    if (err instanceof Error) {
      error.value = err;
    }
    layoutStore.loading = false;
  }
};
const keyEvent = (event: KeyboardEvent) => {
  if (event.key === "F1") {
    event.preventDefault();
    layoutStore.showHover("help");
  }
};
</script>
