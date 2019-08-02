<template>
  <div class="home">
    <div class="home-top">
      <div class="home-top-logo">
        <router-link :to="{name: 'home'}">Moblog</router-link>
      </div>
      <div class="home-top-right">
        Logged in as {{ userLabel }} | 
        <a href="#" v-on:click.prevent="logout">Logout</a>
      </div>
    </div>

    <div class="home-main">
      <router-view/>
    </div>
  </div>
</template>

<style>
div.home {
  flex: 1 1 auto;

  display: flex;
  flex-direction: column;
}

div.home-top {
  flex: 0 0 3rem;

  display: flex;
  flex-direction: row;
  align-items: center;

  box-shadow: 0px 0px 5px 0px rgba(0, 0, 0, 0.50);
  margin-bottom: 1rem;
}

div.home-top-logo {
  margin-left: 1rem;
}

div.home-top-right {
  margin-left: auto;
  margin-right: 1rem;
}

div.home-main {
  flex: 1 1 auto;

  margin: 0 1rem 0 1rem;
  display: flex;
}
</style>

<script lang="ts">
import { Component, Vue, Watch } from 'vue-property-decorator';
import { Action, State } from 'vuex-class';

import * as Types from '@/api-types.ts';
import * as Loadable from '@/loadable.ts';

import BlogList from '@/components/BlogList.vue';

@Component({components: {BlogList}})
export default class Home extends Vue {
  @Action private logout: () => Promise<void>;
  @State private user: Loadable.Data<Types.User>;

  private get userLabel(): string {
    if (this.user.state === Loadable.State.Loaded) {
      return this.user.data.DisplayName || this.user.data.Username;
    } else {
      return '';
    }
  }

  @Watch('user') private onUserChanged(user: Loadable.Data<Types.User>) {
    if (user.state !== Loadable.State.Loaded) {
      this.$router.replace({name: 'login'});
    }
  }
}
</script>
