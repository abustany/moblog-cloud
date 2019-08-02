import * as Types from '@/api-types.ts';

const ApiUrl: string = process.env.API_URL || '/api';

const FakeApiDelay = 500; // ms

export function login(username: string, password: string): Promise<boolean> {
  const body = new URLSearchParams();
  body.append('username', username);
  body.append('password', password);

  return fetch(ApiUrl + '/login', {
    method: 'POST',
    credentials: 'same-origin',
    body,
  }).then((res) => {
    if (res.status === 401) {
      return false;
    }

    if (!res.ok) {
      return Promise.reject('Login request failed with status ' + res.status);
    }

    return true;
  });
}

export function logout(): Promise<void> {
  return fetch(ApiUrl + '/logout', {
    method: 'POST',
    credentials: 'same-origin',
  }).then((res) => {
    if (!res.ok) {
      return Promise.reject('Logout request failed with status ' + res.status);
    }
  });
}

const callId: () => string = (() => {
  let n = 0;

  return () => {
    n++;
    return '' + n;
  };
})();

interface JsonRPCError {
  error: any;
}

interface JsonRPCResult<T> {
  result: T;
}

type JsonRPCResponse<T> = JsonRPCError | JsonRPCResult<T>;

function delayedPromise<T>(promise: Promise<T>, delay: number): Promise<T> {
  if (delay === 0) {
    return promise;
  }

  return new Promise((resolve, reject) => setTimeout(() => resolve(), delay))
    .then(() => promise);
}

function api<T>(method: string, params?: object): Promise<T> {
  const call = {
    id: callId(),
    method,
    params: (params === undefined) ? [] : [
      params,
    ],
  };

  return delayedPromise(fetch(ApiUrl + '/', {
    method: 'POST',
    credentials: 'same-origin',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(call),
  }).then((res) => {
    if (res.status !== 200) {
      return Promise.reject('Invalid response code: ' + res.status);
    }

    return res.json();
  }).then((res) => {
    if (res.error) {
      return Promise.reject(`RPC Error (${method}): ${(res as JsonRPCError).error}`);
    }

    return (res as JsonRPCResult<T>).result;
  }), FakeApiDelay);
}

export const User = {
  whoami: () => api<Types.User>('Users.Whoami'),
  register: (user: Types.UserWithPassword) => api<void>('Users.Create', user),
};

export const Blogs = {
  list: () => api<Types.Blog[] | null>('Blogs.List'),
  create: (blog: Types.Blog) => api<void>('Blogs.Create', blog),
  update: (blog: Types.Blog) => api<void>('Blogs.Update', blog),
  delete: (slug: string) => api<void>('Blogs.Delete', {Slug: slug}),
};
