import { defineStore } from "pinia";

export const useFileStore = defineStore("file", {
  // convert to a function
  state: (): {
    req: Resource | null;
    oldReq: Resource | null;
    reload: boolean;
    selected: number[];
    multiple: boolean;
    isFiles: boolean;
    preselect: string | null;
    contextMenuOpen: boolean;
    isSilentRefresh: boolean;
    autoRefreshIntervalMs: number;
    refreshCycle: number;
  } => ({
    req: null,
    oldReq: null,
    reload: false,
    selected: [],
    multiple: false,
    isFiles: false,
    preselect: null,
    contextMenuOpen: false,
    // взведён на время тихого авто-обновления: подсказка листингу не трогать
    // showLimit и скролл (иначе список схлопывается и страницу кидает наверх)
    isSilentRefresh: false,
    // единственный источник интервала авто-обновления: питает и JS-таймер (Files.vue),
    // и длительность CSS-анимации кольца-«часов» (FileListing.vue, инлайн animation-duration)
    autoRefreshIntervalMs: 5000,
    // счётчик циклов: меняется в начале каждого отсчёта → через :key перезапускает анимацию кольца
    refreshCycle: 0,
  }),
  getters: {
    selectedCount: (state) => state.selected.length,
    // route: () => {
    //   const routerStore = useRouterStore();
    //   return routerStore.router.currentRoute;
    // },
    // isFiles: (state) => {
    //   const layoutStore = useLayoutStore();
    //   return !layoutStore.loading && state.route._value.name === "Files";
    // },
    isListing: (state) => {
      return state.isFiles && state?.req?.isDir;
    },
  },
  actions: {
    // no context as first argument, use `this` instead
    toggleMultiple() {
      this.multiple = !this.multiple;
    },
    updateRequest(value: Resource | null) {
      const selectedItems = this.selected.map((i) => this.req?.items[i]);
      this.oldReq = this.req;
      this.req = value;

      this.selected = [];

      if (!this.req?.items) return;
      this.selected = this.req.items
        .filter((item) =>
          selectedItems.some((rItem) => rItem?.url === item.url)
        )
        .map((item) => item.index);
    },
    removeSelected(value: any) {
      const i = this.selected.indexOf(value);
      if (i === -1) return;
      this.selected.splice(i, 1);
    },
    // easily reset state using `$reset`
    clearFile() {
      this.$reset();
    },
  },
});
