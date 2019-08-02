import Vue from 'vue';
import Vuex, { StoreOptions} from 'vuex';

import * as Types from '@/api-types.ts';
import * as Api from '@/api.ts';
import * as Loadable from '@/loadable.ts';

Vue.use(Vuex);

interface State {
  user: Loadable.Data<Types.User>;
  blogs: Loadable.Data<Types.Blog[]>;
}

interface SetUserMutation {
  type: 'setUser';
  user: Loadable.Data<Types.User>;
}

interface SetBlogsMutation {
  type: 'setBlogs';
  blogs: Loadable.Data<Types.Blog[]>;
}

type Mutation = SetUserMutation | SetBlogsMutation;

export interface RegisterActionParams {
  user: Types.UserWithPassword;
}

interface RegisterAction extends RegisterActionParams {
  type: 'register';
}

export interface LoginActionParams {
  username: string;
  password: string;
}

interface LoginAction extends LoginActionParams {
  type: 'login';
}

interface LogoutAction {
  type: 'logout';
}

interface CheckLoginAction {
  type: 'checkLogin';
}

interface LoadBlogsActions {
  type: 'loadBlogs';
}

export interface CreateBlogActionParams {
  blog: Types.Blog;
}

interface CreateBlogAction extends CreateBlogActionParams {
  type: 'createBlog';
}

export interface EditBlogActionParams {
  blog: Types.Blog;
}

interface EditBlogAction extends EditBlogActionParams {
  type: 'editBlog';
}

export interface DeleteBlogActionParams {
  slug: string;
}

interface DeleteBlogAction extends DeleteBlogActionParams {
  type: 'deleteBlog';
}

type Action =
  RegisterAction |
  LoginAction |
  LogoutAction |
  CheckLoginAction |
  LoadBlogsActions;

type CommitFunc = (m: Mutation) => void;
type DispatchFunc = (a: Action) => Promise<void>;

interface ActionContext {
  commit: CommitFunc;
  dispatch: DispatchFunc;
  state: State;
}

function sortBlogList(list: Types.Blog[]): Types.Blog[] {
  const nameOrSlug = (b: Types.Blog) => (b.DisplayName || b.Slug).toLowerCase();
  const sorted = [...list];
  sorted.sort((a, b) => nameOrSlug(a).localeCompare(nameOrSlug(b)));

  return sorted;
}

function editBlogsHelper<T>(
  {commit, state}: ActionContext,
  localFn: (blogs: Types.Blog[]) => Types.Blog[],
  apiFn: (data: T) => Promise<void>,
  payload: T,
): Promise<void> {
  if (state.blogs.state !== Loadable.State.Loaded) {
    return Promise.reject('Invalid base state');
  }

  const oldBlogs = state.blogs.data;

  commit({ type: 'setBlogs', blogs: Loadable.loading });

  return apiFn(payload).then(() => {
    const newBlogs = sortBlogList(localFn(oldBlogs));
    commit({ type: 'setBlogs', blogs: Loadable.loaded(newBlogs)});
  }, (err) => {
    commit({ type: 'setBlogs', blogs: Loadable.loaded(oldBlogs)});
    return Promise.reject(err);
  });
}

const store: StoreOptions<State> = {
  state: {
    user: Loadable.uninitialized,
    blogs: Loadable.uninitialized,
  },

  // MUTATIONS
  mutations: {
    setUser(state: State, payload: SetUserMutation) {
      if (payload.user.state !== Loadable.State.Loaded ||
          (state.user.state === Loadable.State.Loaded && payload.user.data.Username !== state.user.data.Username)) {
        state.blogs = Loadable.uninitialized;
      }

      state.user = payload.user;
    },

    setBlogs(state: State, payload: SetBlogsMutation) {
      state.blogs = payload.blogs;
    },
  },

  // ACTIONS
  actions: {
    register({ commit, dispatch, state }: ActionContext, payload: RegisterAction): Promise<void> {
      commit({ type: 'setUser', user: Loadable.loading });

      return Api.User.register(payload.user).then(() => {
        return dispatch({
          type: 'login',
          username: payload.user.Username,
          password: payload.user.Password,
        });
      }, (err) => {
        commit({ type: 'setUser', user: Loadable.error(err) });
      });
    },

    login({ commit, dispatch, state }: ActionContext, payload: LoginAction): Promise<void> {
      commit({ type: 'setUser', user: Loadable.loading });

      return Api.login(payload.username, payload.password).then((ok) => {
        if (ok) {
          return dispatch({ type: 'checkLogin' });
        } else {
          commit({type: 'setUser', user: Loadable.error('Invalid username and/or password') });
        }
      }, (err) => {
        commit({ type: 'setUser', user: Loadable.error(err) });
      });
    },

    logout({ commit, state }: ActionContext): Promise<void> {
      commit({ type: 'setUser', user: Loadable.loading });

      return Api.logout().then(() => {
        commit({ type: 'setUser', user: Loadable.uninitialized });
      }, (err) => {
        commit({ type: 'setUser', user: Loadable.error(err) });
      });
    },

    checkLogin({ commit, state }: ActionContext): Promise<void> {
      commit({ type: 'setUser', user: Loadable.loading });

      return Api.User.whoami().then((user) => {
        commit({ type: 'setUser', user: Loadable.loaded(user) });
      }).catch((err) => {
        if (err.indexOf('This method requires authentication') !== -1) {
          // FIXME some kind of error code would be less horrible
          commit({ type: 'setUser', user: Loadable.uninitialized });
        } else {
          commit({ type: 'setUser', user: Loadable.error(err) });
        }
      });
    },

    loadBlogs({ commit, state }: ActionContext): Promise<void> {
      commit({ type: 'setBlogs', blogs: Loadable.loading });

      return Api.Blogs.list().then((blogs) => {
        commit({ type: 'setBlogs', blogs: Loadable.loaded(blogs || []) });
      }, (err) => {
        commit({ type: 'setBlogs', blogs: Loadable.error(err) });
      });
    },

    createBlog(ctx: ActionContext, payload: CreateBlogAction): Promise<void> {
      return editBlogsHelper(
        ctx,
        (blogs) => [...blogs, payload.blog],
        Api.Blogs.create,
        payload.blog,
      );
    },

    editBlog(ctx: ActionContext, payload: EditBlogAction): Promise<void> {
      return editBlogsHelper(
        ctx,
        (blogs) => [...blogs.filter((b) => b.Slug !== payload.blog.Slug), payload.blog],
        Api.Blogs.update,
        payload.blog,
      );
    },

    deleteBlog(ctx: ActionContext, payload: DeleteBlogAction): Promise<void> {
      return editBlogsHelper(
        ctx,
        (blogs) => blogs.filter((b) => b.Slug !== payload.slug),
        Api.Blogs.delete,
        payload.slug,
      );
    },
  },
};

export default new Vuex.Store<State>(store);
