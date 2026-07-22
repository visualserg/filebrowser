<template>
  <div class="card floating">
    <div class="card-title">
      <h2>{{ t("prompts.newDir") }}</h2>
    </div>

    <div class="card-content">
      <p>{{ t("prompts.newDirMessage") }}</p>
      <div class="new-dir-input">
        <input
          id="focus-prompt"
          class="input input--block"
          type="text"
          @keyup.enter="submit"
          v-model.trim="name"
          tabindex="1"
        />
        <button
          class="button button--flat new-dir-input__today"
          type="button"
          @click="fillToday"
          :aria-label="t('buttons.todayDate')"
          :title="t('buttons.todayDate')"
          tabindex="4"
        >
          <i class="material-icons">today</i>
        </button>
      </div>
      <CreateFilePath :name="name" :is-dir="true" :path="base" />
    </div>

    <div class="card-action">
      <button
        class="button button--flat button--grey"
        @click="layoutStore.closeHovers"
        :aria-label="t('buttons.cancel')"
        :title="t('buttons.cancel')"
        tabindex="3"
      >
        {{ t("buttons.cancel") }}
      </button>
      <button
        class="button button--flat"
        :aria-label="$t('buttons.create')"
        :title="t('buttons.create')"
        @click="submit"
        tabindex="2"
      >
        {{ t("buttons.create") }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, ref } from "vue";
import { useFileStore } from "@/stores/file";
import { useLayoutStore } from "@/stores/layout";

import { files as api } from "@/api";
import url from "@/utils/url";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import CreateFilePath from "@/components/prompts/CreateFilePath.vue";

const $showError = inject<IToastError>("$showError")!;

const fileStore = useFileStore();
const layoutStore = useLayoutStore();

const base = computed(() => {
  return layoutStore.currentPrompt?.props?.base;
});

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const name = ref<string>("");

// Подставляет сегодняшнюю дату в формате ГГГГММДД (8 символов, без разделителей).
const fillToday = () => {
  const now = new Date();
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, "0");
  const d = String(now.getDate()).padStart(2, "0");
  name.value = `${y}${m}${d}`;
};

const submit = async (event: Event) => {
  event.preventDefault();
  if (name.value === "") return;

  // Build the path of the new directory.
  let uri: string;
  if (base.value) uri = base.value;
  else if (fileStore.isFiles) uri = route.path + "/";
  else uri = "/";

  if (!fileStore.isListing) {
    uri = url.removeLastDir(uri) + "/";
  }

  uri += encodeURIComponent(name.value) + "/";
  uri = uri.replace("//", "/");

  try {
    await api.post(uri);
    if (layoutStore.currentPrompt?.props?.redirect) {
      router.push({ path: uri });
    } else if (!base.value) {
      const res = await api.fetch(url.removeLastDir(uri) + "/");
      fileStore.updateRequest(res);
    }
    if (layoutStore.currentPrompt?.confirm) {
      layoutStore.currentPrompt?.confirm(uri);
    }
  } catch (e) {
    if (e instanceof Error) {
      $showError(e);
    }
  }

  layoutStore.closeHovers();
};
</script>

<style scoped>
.new-dir-input {
  display: flex;
  align-items: center;
  gap: 0.5em;
}

.new-dir-input .input {
  flex: 1 1 auto;
}

.new-dir-input__today {
  flex: 0 0 auto;
  margin: 0;
  padding: 0.35em;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.new-dir-input__today i {
  font-size: 1.25em;
}
</style>
