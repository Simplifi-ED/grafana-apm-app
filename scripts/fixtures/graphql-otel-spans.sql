-- CI / local smoke fixture for GraphQL span detection (ticket 16).
-- Insert into a test ClickHouse with ClickStack otel_traces schema
-- (columns match docs/clickstack-migration/schema.md / DESCRIBE otel_traces).
-- Adjust Timestamp / ServiceName as needed for your seed window.
--
-- Verify:
--   SELECT count() FROM otel_traces
--   WHERE mapContains(SpanAttributes, 'graphql.operation.name');

INSERT INTO otel_traces (
  Timestamp,
  TraceId,
  SpanId,
  ParentSpanId,
  TraceState,
  SpanName,
  SpanKind,
  ServiceName,
  ResourceAttributes,
  ScopeName,
  ScopeVersion,
  SpanAttributes,
  Duration,
  StatusCode,
  StatusMessage,
  `Events.Timestamp`,
  `Events.Name`,
  `Events.Attributes`,
  `Links.TraceId`,
  `Links.SpanId`,
  `Links.TraceState`,
  `Links.Attributes`
) VALUES
(
  now64(9),
  'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
  'bbbbbbbbbbbbbbbb',
  '',
  '',
  'query GetInvoice',
  'Server',
  'graphql-demo',
  map(
    'service.name', 'graphql-demo',
    'service.namespace', 'demo',
    'deployment.environment', 'ci',
    'cluster', 'ci'
  ),
  'graphql-demo',
  '0.0.1',
  map(
    'graphql.operation.name', 'GetInvoice',
    'graphql.operation.type', 'query',
    'graphql.document', 'query GetInvoice { invoice(id: $id) { id } }'
  ),
  toUInt64(12500000),
  'Ok',
  '',
  [],
  [],
  [],
  [],
  [],
  [],
  []
),
(
  now64(9),
  'cccccccccccccccccccccccccccccccc',
  'dddddddddddddddd',
  '',
  '',
  'mutation CreateInvoice',
  'Server',
  'graphql-demo',
  map(
    'service.name', 'graphql-demo',
    'service.namespace', 'demo',
    'deployment.environment', 'ci',
    'cluster', 'ci'
  ),
  'graphql-demo',
  '0.0.1',
  map(
    'graphql.operation.name', 'CreateInvoice',
    'graphql.operation.type', 'mutation'
  ),
  toUInt64(42000000),
  'Error',
  'resolver failed',
  [],
  [],
  [],
  [],
  [],
  [],
  []
);
