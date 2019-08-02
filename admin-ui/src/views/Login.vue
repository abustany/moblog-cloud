<template>
  <div class="login">
    <form v-on:submit.prevent="onSubmit">
      <fieldset :disabled="loading">
        <label>Username <input type="text" v-model="username"/></label>
        <label>Password <input type="password" v-model="password"/></label>
      </fieldset>
      <input type="submit" :value="submitLabel" :disabled="loading"/>
      <p v-if="error">{{ error }}</p>
      <p>
        <router-link :to="{name:'register'}">Don't have an account? Register!</router-link>
      </p>
    </form>
  </div>
</template>

<style>
div.login {
  flex: 1 1 auto;

  display: flex;
  align-items: center;
  justify-content: center;
}

.login form {
  text-align: center;
}

.login fieldset {
  border: 0;
}

.login label {
  display: block;
  width: 100%;
  text-align: right;
  margin-bottom: .3rem;
}

.login input[type="text"], .login input[type="password"] {
  width: 20rem;
  margin-left: .3rem;
}

.login input[type="submit"] {
  width: 15rem;
  text-align: center;
}
</style>

<script lang="ts">
import { Component, Prop, Vue, Watch } from 'vue-property-decorator';
import { Action, State } from 'vuex-class';
import Router from 'vue-router';

import * as Api from '@/api.ts';
import * as Types from '@/api-types.ts';
import * as Loadable from '@/loadable.ts';
import { LoginActionParams } from '@/store.ts';

@Component
export default class Login extends Vue {
  @Prop(String) private returnTo: string;
  @Action private login: (params: LoginActionParams) => Promise<void>;
  @Action private checkLogin: () => Promise<void>;
  @State private user: Loadable.Data<Types.User>;

  private username = '';
  private password = '';
  private error: string | null = null;

  public mounted() {
    console.log('will return to', this.returnTo);
    this.checkLogin();
  }

  private get loading() {
    return this.user.state === Loadable.State.Loading;
  }

  private get submitLabel(): string {
    return this.loading ? 'Logging in…' : 'Log in';
  }

  @Watch('user') private onUserChanged(user: Loadable.Data<Types.User>) {
    if (user.state === Loadable.State.Loaded) {
      this.$router.replace(this.returnTo || {name: 'home'});
    }

    if (user.state === Loadable.State.Error) {
      this.error = user.error;
    }
  }

  private onSubmit() {
    this.login({username: this.username, password: this.password});
  }
}
</script>
