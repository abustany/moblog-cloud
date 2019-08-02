<template>
  <div class="edit-blog">
    <h1 v-if="!!editSlug">Edit blog</h1>
    <h1 v-else>New blog</h1>

    <form v-on:submit.prevent="onSubmit">
        <label>Name <input type="text" spellcheck="false" v-model="name" required/></label>
        <p class="edit-blog-hint">The name that will be displayed on your blog</p>

        <template v-if="!!!editSlug">
          <label>Slug <input type="text" spellcheck="false" :pattern="validSlugRe" v-model="slug"/></label>
          <p class="edit-blog-hint">The slug is a unique identifier for your blog, used in its URL. It must start with a letter or a digit and can include letters, digits, _ - and .</p>
        </template>

        <p v-if="error" class="edit-blog-error">{{ error }}</p>

        <input type="submit" :value="submitLabel" :disabled="loading"/>

        <p v-if="!!editSlug" class="edit-blog-danger-zone">
          <input type="button" value="Delete blog" v-on:click="onDeleteBlog"/>
        </p>
    </form>
  </div>
</template>

<style>
.edit-blog-hint {
  color: #999;
  font-size: small;
  width: 21rem;
  text-align: left;
  margin: .3rem 0 .3rem auto;
}

.edit-blog-error {
  color: #aa0000;
  width: 24rem;
  margin-left: auto;
  margin-right: auto;
}

.edit-blog form {
  text-align: center;
}

.edit-blog label {
  display: block;
  width: 100%;
  text-align: right;
  margin-bottom: .3rem;
}

.edit-blog input[type="text"], .register input[type="password"] {
  width: 20rem;
  margin-left: .3rem;
}

.edit-blog input[type="submit"] {
  width: 15rem;
  text-align: center;
}
</style>

<script lang="ts">
import { Component, Prop, Vue, Watch } from 'vue-property-decorator';
import { Action, State } from 'vuex-class';

import * as Types from '@/api-types.ts';
import * as Loadable from '@/loadable.ts';
import { CreateBlogActionParams, EditBlogActionParams, DeleteBlogActionParams } from '@/store.ts';

@Component
export default class EditBlog extends Vue {
  @Action private loadBlogs: () => Promise<void>;
  @Action private createBlog: (params: CreateBlogActionParams) => Promise<void>;
  @Action private deleteBlog: (params: DeleteBlogActionParams) => Promise<void>;
  @Action private editBlog: (params: EditBlogActionParams) => Promise<void>;
  @Prop(String) private editSlug: string | undefined;
  @State private blogs: Loadable.Data<Types.Blog[]>;

  private validSlugRe = '[a-zA-Z0-9][a-zA-Z0-9\\-_\\.]*';
  private name = '';
  private slug = '';
  private error: string | null = null;
  private deleted = false;

  public mounted() {
    if (this.blogs.state === Loadable.State.Uninitialized) {
      this.loadBlogs();
    }

    if (this.blogs.state === Loadable.State.Loaded) {
      this.populateForm();
    } // else, onBlogsChanged will do it
  }

  @Watch('blogs') private onBlogsChanged(blogs: Loadable.Data<Types.Blog[]>) {
    if (blogs.state === Loadable.State.Loaded && !this.deleted) {
      this.populateForm();
    }
  }

  private get loading(): boolean {
    return this.blogs.state === Loadable.State.Loading;
  }

  private get submitLabel() {
    if (this.loading) {
      return 'Working…';
    }

    return this.editSlug ? 'Save' : 'Create';
  }

  private populateForm() {
    if (!this.editSlug || this.blogs.state !== Loadable.State.Loaded) {
      return;
    }

    const blog = this.blogs.data.find((b) => b.Slug === this.editSlug);

    if (!blog) {
      console.log(`Couldn't find a blog with slug ${this.editSlug}?!`);
      return;
    }

    this.name = blog.DisplayName || '';
    this.slug = blog.Slug;
  }

  private nameToSlug(name: string): string {
    return name.
      trim().
      replace(/[^a-zA-Z0-9\-\._]/g, '-').
      replace(/^-+/, '').
      replace(/-+$/, '');
  }

  @Watch('name') private onNameChanged(newName: string, oldName: string) {
    if (this.editSlug) {
      return;
    }

    if (this.slug === this.nameToSlug(oldName)) {
      this.slug = this.nameToSlug(newName);
    }
  }

  @Watch('slug') private onSlugChanged(slug: string) {
    const blogList = (this.blogs.state === Loadable.State.Loaded ? this.blogs.data : []);

    if (!this.editSlug) {
      // For new blogs, check slug uniqueness
      const sameSlugBlog = blogList.find((b) => b.Slug === slug);

      if (sameSlugBlog) {
        this.error = `A blog named "${sameSlugBlog.DisplayName}"
          with the same slug (${sameSlugBlog.Slug}) already exists`;
      } else {
        this.error = null;
      }
    }
  }

  private validate(): string | null {
    if (this.error !== '') {
      return this.error;
    }

    if (this.name.trim() === '') {
      return 'Blog name cannot be empty';
    }

    if (this.slug.trim() === '') {
      return 'Slug cannot be empty';
    }

    return null;
  }

  private onSubmit() {
    this.error = this.validate();

    if (this.error) {
      return;
    }

    const blog: Types.Blog = {
      Slug: this.editSlug || this.slug.trim(),
      DisplayName: this.name.trim(),
    };

    const action = this.editSlug ? this.editBlog : this.createBlog;

    action({blog}).then(() => {
      this.$router.replace({name: 'home'});
    }).catch((err: string) => {
      this.error = err;
    });
  }

  private onDeleteBlog() {
    if (!this.editSlug) {
      // Just in case...
      return;
    }

    if (!confirm('All blog contents will be deleted! This action cannot be undone.')) {
      return;
    }

    this.deleted = true;

    this.deleteBlog({slug: this.editSlug}).then(() => {
      this.$router.replace({name: 'home'});
    }).catch((err: string) => {
      this.deleted = false;
      this.error = err;
    });
  }
}
</script>
