import React from 'react';
import { Alert } from '@grafana/ui';
import { TableSkeleton } from './Skeleton';

interface DataStateProps {
  loading?: boolean;
  error?: string | null;
  empty?: boolean;
  /**
   * @deprecated Prefer `skeleton`. Ignored when loading — DataState defaults to TableSkeleton.
   */
  loadingText?: string;
  /** When set, shown instead of the default TableSkeleton while loading. */
  skeleton?: React.ReactNode;
  errorTitle?: string;
  emptyTitle?: string;
  emptyMessage?: React.ReactNode;
  children: React.ReactNode;
}

/**
 * Handles the common loading → error → empty → content state machine.
 * Renders the first matching state; `children` is rendered only when
 * not loading, no error, and not empty.
 */
export function DataState({
  loading,
  error,
  empty,
  skeleton,
  errorTitle = 'Error',
  emptyTitle = 'No data',
  emptyMessage = 'No data found.',
  children,
}: DataStateProps) {
  if (error) {
    return (
      <Alert severity="error" title={errorTitle}>
        {error}
      </Alert>
    );
  }
  if (loading) {
    return <>{skeleton ?? <TableSkeleton aria-label="Loading data" />}</>;
  }
  if (empty) {
    return (
      <Alert severity="info" title={emptyTitle}>
        {emptyMessage}
      </Alert>
    );
  }
  return <>{children}</>;
}
