<template>
  <div class="register">
    <form v-on:submit.prevent="onSubmit">
      <fieldset :disabled="loading">
        <label>Username <input type="text" spellcheck="false" :pattern="validUsernameRe" v-model="username"/></label>
        <p class="register-hint">Usernames must start with a letter, and can include letters, digits, and the symbols _ - and .</p>
        <label>Display name <input type="text" v-model="displayName"/></label>
        <p class="register-hint">(Optional) The name that will be shown publicly to your readers</p>
        <label>Password <input type="password" :minlength="minPasswordLength" v-model="password"/></label>
        <p class="register-hint">Password must be at least four character long</p>
        <label>Password (confirm)<input type="password" minlength="4" v-model="passwordConfirm"/></label>
      </fieldset>
      <input type="submit" :value="submitLabel" :disabled="loading"/>
      <p v-if="error">{{ error }}</p>
      <p>
        <router-link :to="{name:'login'}">Already have an account? Log in instead!</router-link>
      </p>
    </form>
  </div>
</template>

<style>
div.register {
  flex: 1 1 auto;

  display: flex;
  align-items: center;
  justify-content: center;
}

.register-hint {
  color: #999;
  font-size: small;
  width: 21rem;
  text-align: left;
  margin: .3rem 0 .3rem auto;
}

.register form {
  text-align: center;
}

.register fieldset {
  border: 0;
}

.register label {
  display: block;
  width: 100%;
  text-align: right;
  margin-bottom: .3rem;
}

.register input[type="text"], .register input[type="password"] {
  width: 20rem;
  margin-left: .3rem;
}

.register input[type="submit"] {
  width: 15rem;
  text-align: center;
}
</style>

<script lang="ts">
import { Component, Inject, Vue, Watch } from 'vue-property-decorator';
import { Action, State } from 'vuex-class';

import * as Types from '@/api-types.ts';
import * as Loadable from '@/loadable.ts';
import { RegisterActionParams } from '@/store.ts';

@Component
export default class Register extends Vue {
  @Action private register: (params: RegisterActionParams) => Promise<void>;
  @State private user: Loadable.Data<Types.User>;

  private validUsernameRe = '[a-zA-Z][a-zA-Z0-9\\-_\\.]+';
  private minPasswordLength = 4;
  private username = '';
  private displayName = '';
  private password = '';
  private passwordConfirm = '';
  private error: string | null = null;

  private get loading() {
    return this.user.state === Loadable.State.Loading;
  }

  private validate(): string | null {
    if (this.username === '') {
      return 'Username cannot be empty';
    }

    if (!this.username.match(RegExp('^' + this.validUsernameRe + '$'))) {
      return 'Username contains invalid characters';
    }

    if (this.password.length < this.minPasswordLength) {
      return `Password should be longer than ${this.minPasswordLength} characters`;
    }

    if (this.password !== this.passwordConfirm) {
      return 'Password fields don\'t match';
    }

    return null;
  }

  private get submitLabel(): string {
    return this.loading ? 'Registering user…' : 'Register';
  }

  @Watch('user') private onUserChanged(user: Loadable.Data<Types.User>) {
    if (user.state === Loadable.State.Loaded) {
      this.$router.replace({name: 'home'});
    }

    if (user.state === Loadable.State.Error) {
      this.error = user.error;
    }
  }

  private onSubmit() {
    this.error = this.validate();

    if (this.error) {
      return;
    }

    const user: Types.UserWithPassword = {
      Username: this.username,
      DisplayName: this.displayName,
      Password: this.password,
    };

    this.register({user});
  }
}
</script>
