<template>
  <div
    id="modal-background"
    @mousedown="backgroundMouseDown"
    @mouseup="backgroundMouseUp"
  >
    <div ref="modalContainer">
      <slot></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";

const emit = defineEmits(["closed"]);

const modalContainer = ref(null);

onMounted(() => {
  const element = document.querySelector("#focus-prompt") as HTMLElement | null;
  if (element) {
    element.focus();
  } else if (modalContainer.value) {
    (modalContainer.value as HTMLElement).focus();
  }
});

// Закрываем только если и нажатие, и отпускание пришлись на фон: иначе
// выделение текста мышкой, уехавшее за пределы формы, закрывало бы модалку.
const pressedOnBackground = ref(false);

const isBackground = (event: MouseEvent) =>
  (event.target as HTMLElement).id === "modal-background";

const backgroundMouseDown = (event: MouseEvent) => {
  pressedOnBackground.value = isBackground(event);
};

const backgroundMouseUp = (event: MouseEvent) => {
  const closing = pressedOnBackground.value && isBackground(event);
  pressedOnBackground.value = false;
  if (closing) {
    emit("closed");
  }
};

window.addEventListener("keydown", (event) => {
  if (event.key === "Escape") {
    event.stopImmediatePropagation();
    emit("closed");
  }
});
</script>

<style scoped>
#modal-background {
  position: fixed;
  inset: 0;
  background-color: #00000096;
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 10000;
  animation: ease-in 150ms opacity-enter;
}

@keyframes opacity-enter {
  from {
    opacity: 0;
  }

  to {
    opacity: 1;
  }
}
</style>
