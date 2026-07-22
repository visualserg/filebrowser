<template>
  <div class="breadcrumbs">
    <component
      :is="element"
      :to="base || ''"
      :aria-label="t('files.home')"
      :title="t('files.home')"
      :class="{ 'drop-target': dragOver === 'home' }"
      @dragover="onDragOver($event, homeDest)"
      @dragenter="onDragEnter($event, homeDest, 'home')"
      @dragleave="onDragLeave('home')"
      @drop="onDrop($event, homeDest)"
    >
      <i class="material-icons">home</i>
    </component>

    <span v-for="(link, index) in items" :key="index">
      <span class="chevron"
        ><i class="material-icons">keyboard_arrow_right</i></span
      >
      <component
        :is="element"
        :to="link.url"
        :class="{ 'drop-target': dragOver === index }"
        @dragover="onDragOver($event, link.url)"
        @dragenter="onDragEnter($event, link.url, index)"
        @dragleave="onDragLeave(index)"
        @drop="onDrop($event, link.url)"
        >{{ link.name }}</component
      >
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { useFileStore } from "@/stores/file";
import { useAuthStore } from "@/stores/auth";
import { useMoveSelected } from "@/composables/useMoveSelected";

const { t } = useI18n();

const route = useRoute();
const fileStore = useFileStore();
const authStore = useAuthStore();
const { moveSelectedTo } = useMoveSelected();

const props = defineProps<{
  base: string;
  noLink?: boolean;
}>();

// Какая крошка сейчас под курсором при перетаскивании ('home' | индекс сегмента).
const dragOver = ref<number | "home" | null>(null);

// Папка-назначение для иконки дома (корень base).
const homeDest = computed(() =>
  props.base.endsWith("/") ? props.base : props.base + "/"
);

// Можно ли бросить выбранное в папку dest: только в обычном (не read-only)
// режиме, при праве rename, наличии выделения и если это не текущая же папка.
const canAcceptDrop = (dest: string): boolean => {
  if (props.noLink) return false;
  if (!authStore.user?.perm.rename) return false;
  if (fileStore.selectedCount === 0) return false;
  const cur = route.path.endsWith("/") ? route.path : route.path + "/";
  return dest !== cur;
};

const onDragOver = (event: DragEvent, dest: string) => {
  if (!canAcceptDrop(dest)) return;
  event.preventDefault(); // разрешаем drop именно сюда
};

const onDragEnter = (event: DragEvent, dest: string, key: number | "home") => {
  if (!canAcceptDrop(dest)) return;
  event.preventDefault();
  dragOver.value = key;
};

const onDragLeave = (key: number | "home") => {
  if (dragOver.value === key) dragOver.value = null;
};

const onDrop = (event: DragEvent, dest: string) => {
  dragOver.value = null;
  if (!canAcceptDrop(dest)) return;
  // не даём событию улететь в document-обработчик загрузки файлов из ОС
  event.preventDefault();
  event.stopPropagation();
  moveSelectedTo(dest, event.ctrlKey || event.metaKey);
};

const items = computed(() => {
  const relativePath = route.path.replace(props.base, "");
  const parts = relativePath.split("/");

  if (parts[0] === "") {
    parts.shift();
  }

  if (parts[parts.length - 1] === "") {
    parts.pop();
  }

  const breadcrumbs: BreadCrumb[] = [];

  for (let i = 0; i < parts.length; i++) {
    if (i === 0) {
      breadcrumbs.push({
        name: decodeURIComponent(parts[i]),
        url: props.base + "/" + parts[i] + "/",
      });
    } else {
      breadcrumbs.push({
        name: decodeURIComponent(parts[i]),
        url: breadcrumbs[i - 1].url + parts[i] + "/",
      });
    }
  }

  if (breadcrumbs.length > 3) {
    while (breadcrumbs.length !== 4) {
      breadcrumbs.shift();
    }

    breadcrumbs[0].name = "...";
  }

  return breadcrumbs;
});

const element = computed(() => {
  if (props.noLink) {
    return "span";
  }

  return "router-link";
});
</script>

<style scoped>
/* Подсветка крошки-цели, когда на неё перетаскивают файл */
.breadcrumbs .drop-target {
  background-color: rgba(0, 0, 0, 0.12);
  border-radius: 4px;
  outline: 2px dashed currentColor;
  outline-offset: -2px;
}
</style>
