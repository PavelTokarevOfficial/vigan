<script setup lang="ts">
defineProps<{ message: string }>()
</script>

<template>
  <p class="error-state" role="alert">{{ message }}</p>
</template>
