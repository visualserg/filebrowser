<template>
  <div class="card floating">
    <div class="card-title">
      <h2>{{ $t("prompts.rename") }}</h2>
    </div>

    <div class="card-content">
      <input
        id="focus-prompt"
        ref="input"
        class="input input--block"
        type="text"
        @keyup.enter="submit"
        v-model.trim="name"
      />
    </div>

    <div class="card-action">
      <button
        class="button button--flat button--grey"
        @click="closeHovers"
        :aria-label="$t('buttons.cancel')"
        :title="$t('buttons.cancel')"
      >
        {{ $t("buttons.cancel") }}
      </button>
      <button
        @click="submit"
        class="button button--flat"
        type="submit"
        :aria-label="$t('buttons.rename')"
        :title="$t('buttons.rename')"
        :disabled="name === '' || name === oldName"
      >
        {{ $t("buttons.rename") }}
      </button>
    </div>
  </div>
</template>

<script>
import { mapActions, mapState, mapWritableState } from "pinia";
import { useFileStore } from "@/stores/file";
import { useLayoutStore } from "@/stores/layout";
import url from "@/utils/url";
import { files as api } from "@/api";
import { removePrefix } from "@/api/utils";

export default {
  name: "rename",
  data: function () {
    return {
      name: "",
    };
  },
  created() {
    this.name = this.oldName;
  },
  mounted() {
    // BaseModal фокусирует поле уже после нашего mounted, поэтому ставим
    // курсор на следующем тике: у файла перед расширением, иначе в конец.
    this.$nextTick(() => {
      const input = this.$refs.input;
      const dot = this.isDir ? -1 : this.name.lastIndexOf(".");
      const pos = dot > 0 ? dot : this.name.length;
      input.focus();
      input.setSelectionRange(pos, pos);
    });
  },
  inject: ["$showError"],
  computed: {
    ...mapState(useFileStore, [
      "req",
      "selected",
      "selectedCount",
      "isListing",
    ]),
    ...mapWritableState(useFileStore, ["reload", "preselect"]),
    oldName() {
      if (!this.isListing) {
        return this.req.name;
      }

      if (this.selectedCount === 0 || this.selectedCount > 1) {
        // This shouldn't happen.
        return "";
      }

      return this.req.items[this.selected[0]].name;
    },
    isDir() {
      if (!this.isListing) {
        return this.req.isDir;
      }

      return this.req.items[this.selected[0]]?.isDir ?? false;
    },
  },
  methods: {
    ...mapActions(useLayoutStore, ["closeHovers"]),
    cancel: function () {
      this.closeHovers();
    },
    submit: async function () {
      if (this.name === "" || this.name === this.oldName) {
        return;
      }
      let oldLink = "";
      let newLink = "";

      if (!this.isListing) {
        oldLink = this.req.url;
      } else {
        oldLink = this.req.items[this.selected[0]].url;
      }

      newLink =
        url.removeLastDir(oldLink) + "/" + encodeURIComponent(this.name);

      try {
        await api.move([{ from: oldLink, to: newLink }]);
        if (!this.isListing) {
          this.$router.push({ path: newLink });
          return;
        }

        this.preselect = removePrefix(newLink);

        this.reload = true;
      } catch (e) {
        this.$showError(e);
      }

      this.closeHovers();
    },
  },
};
</script>
