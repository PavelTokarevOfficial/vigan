import { createRouter, createWebHistory } from 'vue-router'
import StreamersPage from '../pages/StreamersPage.vue'

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/streamers' },
    { path: '/streamers', component: StreamersPage },
    { path: '/search', component: () => import('../pages/ClipsPage.vue') },
    { path: '/clips', redirect: '/search' },
    { path: '/pipeline', component: () => import('../pages/PipelinePage.vue') },
    { path: '/assets', component: () => import('../pages/AssetsPage.vue') },
    {
      path: '/templates',
      component: () => import('../pages/TemplatesPage.vue'),
    },
    {
      path: '/templates/:id',
      component: () => import('../pages/TemplateEditorPage.vue'),
    },
  ],
})
