import { inject } from "vue";
import { useFileStore } from "@/stores/file";
import { useLayoutStore } from "@/stores/layout";
import { files as api } from "@/api";
import * as upload from "@/utils/upload";

// Общая логика drag-and-drop перемещения: переносит (или копирует) выбранные
// файлы в папку-назначение с разрешением конфликтов имён. Один и тот же код
// используется при бросании файла на папку в списке (ListingItem) и на хлебные
// крошки (Breadcrumbs), чтобы не плодить дубль.
export function useMoveSelected() {
  const fileStore = useFileStore();
  const layoutStore = useLayoutStore();
  const $showError = inject<IToastError>("$showError")!;

  // destUrl - url папки назначения (нормализуем до вида ".../").
  // copy - копировать вместо перемещения (зажат Ctrl/Cmd).
  const moveSelectedTo = async (destUrl: string, copy = false) => {
    if (fileStore.req === null || fileStore.selectedCount === 0) return;

    const dest = destUrl.endsWith("/") ? destUrl : destUrl + "/";

    const items: any[] = [];
    for (const i of fileStore.selected) {
      const it = fileStore.req.items[i];
      items.push({
        from: it.url,
        to: dest + encodeURIComponent(it.name),
        name: it.name,
        size: it.size,
        isDir: it.isDir,
        modified: it.modified,
        overwrite: false,
        rename: false,
      });
    }
    if (items.length === 0) return;

    const run = (overwrite?: boolean, rename?: boolean) => {
      const fn = copy ? api.copy : api.move;
      fn(items, overwrite, rename)
        .then(() => {
          fileStore.reload = true;
        })
        .catch($showError);
    };

    const conflict = await upload.checkConflict(items, dest, true);

    if (conflict.length > 0) {
      layoutStore.showHover({
        prompt: "resolve-conflict",
        props: { conflict },
        confirm: (event: Event, result: Array<ConflictingResource>) => {
          event.preventDefault();
          layoutStore.closeHovers();
          for (let i = result.length - 1; i >= 0; i--) {
            const item = result[i];
            if (item.checked.length == 2) {
              items[item.index].rename = true;
            } else if (
              item.checked.length == 1 &&
              item.checked[0] == "origin"
            ) {
              items[item.index].overwrite = true;
            } else {
              items.splice(item.index, 1);
            }
          }
          if (items.length > 0) run();
        },
      });
      return;
    }

    run(false, false);
  };

  return { moveSelectedTo };
}
