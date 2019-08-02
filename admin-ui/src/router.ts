import Vue from 'vue';
import Router from 'vue-router';

import BlogList from '@/components/BlogList.vue';
import EditBlog from '@/components/EditBlog.vue';
import Home from '@/views/Home.vue';
import Login from '@/views/Login.vue';
import Register from '@/views/Register.vue';

import * as Loadable from '@/loadable.ts';
import Store from '@/store.ts';

Vue.use(Router);

const router = new Router({
  mode: 'history',
  base: process.env.BASE_URL,
  routes: [
    {
      path: '/login',
      name: 'login',
      component: Login,
      props: (route) => ({ returnTo: route.query.returnTo }),
    },
    {
      path: '/register',
      name: 'register',
      component: Register,
    },
    {
      path: '/',
      component: Home,
      meta: {
        requiresLogin: true,
      },
      children: [
        {
          path: '',
          name: 'home',
          component: BlogList,
        },
        {
          path: 'create',
          name: 'createBlog',
          component: EditBlog,
        },
        {
          path: 'edit/:slug',
          name: 'editBlog',
          component: EditBlog,
          props: (route) => ({ editSlug: route.params.slug }),
        },
      ],
    },
  ],
});

router.beforeEach((to, from, next) => {
  if (to.matched.some((r) => r.meta.requiresLogin) && Store.state.user.state !== Loadable.State.Loaded) {
    next({name: 'login', query: {returnTo: to.path}});
  } else {
    next();
  }
});

export default router;
