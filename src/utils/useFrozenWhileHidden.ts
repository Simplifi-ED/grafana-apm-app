import { useState } from 'react';

/**
 * While `active` is true, returns the latest `value`. While hidden, returns the
 * value last seen while active — so dependents (e.g. SceneQueryRunner builders)
 * do not rebuild when a parent time range changes for an inactive tab.
 */
export function useFrozenWhileHidden<T>(value: T, active: boolean): T {
  const [frozen, setFrozen] = useState(value);
  // Adjust state during render (React-supported) when the tab is active so
  // the frozen snapshot tracks the live value without an effect lag.
  if (active && !Object.is(frozen, value)) {
    setFrozen(value);
  }
  return active ? value : frozen;
}
