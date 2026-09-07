<script setup lang="ts">
withDefaults(defineProps<{ variant?: 'primary' | 'danger' | 'secondary'; disabled?: boolean }>(), { variant: 'primary', disabled: false })
</script>

<template>
  <button :disabled="disabled" :class="['app-button', `app-button--${variant}`]">
    <slot />
  </button>
</template>
