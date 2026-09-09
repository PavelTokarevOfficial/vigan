import { createRouter, createWebHistory } from 'vue-router'
import StreamersPage from '../pages/StreamersPage.vue'

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/streamers' },
    { path: '/streamers', component: StreamersPage },
    { path: '/clips', component: () => import('../pages/ClipsPage.vue') },
    { path: '/pipeline', component: () => import('../pages/PipelinePage.vue') },
    { path: '/videos', component: () => import('../pages/VideosPage.vue') },
  ],
})
