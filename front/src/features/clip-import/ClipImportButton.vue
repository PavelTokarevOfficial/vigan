<script setup lang="ts">
import AppButton from '../../shared/ui/AppButton.vue'

defineProps<{ saved: boolean; busy: boolean }>()
defineEmits<{ save: [] }>()
</script>

<template>
  <AppButton :disabled="saved || busy" @click="$emit('save')">
    {{ saved ? 'В избранном' : 'В избранное' }}
  </AppButton>
</template>
