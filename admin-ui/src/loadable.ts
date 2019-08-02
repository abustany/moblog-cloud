export const enum State {
  Uninitialized,
  Loading,
  Loaded,
  Error,
}

interface Uninitialized {
  state: State.Uninitialized;
}

interface Loading {
  state: State.Loading;
}

interface Loaded<T> {
  state: State.Loaded;
  data: T;
}

interface Error {
  state: State.Error;
  error: string;
}

export const uninitialized: Uninitialized = { state: State.Uninitialized };

export const loading: Loading = { state: State.Loading };

export function loaded<T>(data: T): Loaded<T> {
  return { state: State.Loaded, data };
}

export function error(err: string): Error {
  return { state: State.Error, error: err };
}

export type Data<T> = Uninitialized | Loading | Loaded<T> | Error;
