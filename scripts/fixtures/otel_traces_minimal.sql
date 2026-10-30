-- Minimal otel_traces schema for CI / local fixture loads.
-- Column set matches ClickStack DESCRIBE default.otel_traces.
-- Not a full ClickStack install — enough to INSERT + probe GraphQL span attrs.

CREATE TABLE IF NOT EXISTS otel_traces
(
  Timestamp DateTime64(9),
  TraceId String,
  SpanId String,
  ParentSpanId String,
  TraceState String,
  SpanName LowCardinality(String),
  SpanKind LowCardinality(String),
  ServiceName LowCardinality(String),
  ResourceAttributes Map(LowCardinality(String), String),
  ScopeName String,
  ScopeVersion String,
  SpanAttributes Map(LowCardinality(String), String),
  Duration UInt64,
  StatusCode LowCardinality(String),
  StatusMessage String,
  `Events.Timestamp` Array(DateTime64(9)),
  `Events.Name` Array(LowCardinality(String)),
  `Events.Attributes` Array(Map(LowCardinality(String), String)),
  `Links.TraceId` Array(String),
  `Links.SpanId` Array(String),
  `Links.TraceState` Array(String),
  `Links.Attributes` Array(Map(LowCardinality(String), String))
)
ENGINE = MergeTree
ORDER BY (ServiceName, Timestamp);
