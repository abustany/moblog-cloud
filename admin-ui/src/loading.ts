export type Func = <X>(p: Promise<X>) => Promise<X>;

export function loadingWhile<T, K extends keyof T>(setter: (v: boolean) => void): <X>(p: Promise<X>) => Promise<X> {
  return <X>(p: Promise<X>) => {
    setter(true);
    p.then(() => { setter(false); }, () => { setter(false); });
    return p;
  };
}
