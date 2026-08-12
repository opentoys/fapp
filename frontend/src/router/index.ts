import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: () => import('../views/Home.vue') },
    { path: '/app/:id', component: () => import('../views/AppDetail.vue') },
    { path: '/login', component: () => import('../views/Login.vue') },
    {
      path: '/admin',
      component: () => import('../views/admin/Admin.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/app/:id',
      component: () => import('../views/admin/AdminApp.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/upload',
      component: () => import('../views/admin/Upload.vue'),
      meta: { requiresAuth: true },
    },
  ],
})

router.beforeEach((to) => {
  if (to.meta.requiresAuth && !localStorage.getItem('token')) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.path === '/login' && localStorage.getItem('token')) return '/admin'
})

export default router