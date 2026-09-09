import { createApp } from 'vue'
import { createPinia } from 'pinia'
import VueKonva from 'vue-konva'
import App from './App.vue'
import router from './router'
import '../shared/styles.css'

createApp(App)
.use(createPinia())
.use(VueKonva)
.use(router)
.mount('#app')
