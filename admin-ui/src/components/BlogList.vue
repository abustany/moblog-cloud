<template>
  <div class="blog-list">
    <p v-if="loading">Loading...</p>
    <template v-else>
      <h1>Your blogs</h1>

      <template v-if="blogList">
        <ul>
          <li v-for="blog in blogList">
            <router-link :to="{name: 'editBlog', params: {slug: blog.Slug}}">
              {{ blog.DisplayName || blog.Slug }}
            </router-link>
          </li>
        </ul>
        <p>
          Starting a new trip? <router-link :to="{name: 'createBlog'}">Create a new blog</router-link>!
        </p>
      </template>
      <p v-else>
        No blogs yet... Time to <router-link :to="{name: 'createBlog'}">create a new one</router-link>!
      </p>
    </template>
  </div>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import { Action, State } from 'vuex-class';

import * as Types from '@/api-types.ts';
import * as Loadable from '@/loadable.ts';

@Component
export default class BlogList extends Vue {
  @Action private loadBlogs: () => Promise<void>;
  @State private blogs: Loadable.Data<Types.Blog[]>;

  public mounted() {
    if (this.blogs.state === Loadable.State.Uninitialized) {
      this.loadBlogs();
    }
  }

  private get loading(): boolean {
    return this.blogs.state === Loadable.State.Loading;
  }

  private get blogList(): Types.Blog[] | null {
    if (this.blogs.state !== Loadable.State.Loaded || this.blogs.data.length === 0) {
      return null;
    }

    return this.blogs.data;
  }
}
</script>
