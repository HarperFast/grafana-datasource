# pkg/plugin

## Analytics queries always carry a bounded time window

Invariant: every `get_analytics` request leaves with `0 < start_time <= end_time`. Grafana expands `${__from}`/`${__to}` only in the browser; alert and recording rules are evaluated on the server, so the backend receives those macros verbatim, or no bounds at all when a rule author removes the fields. The SDK omits a zero bound on the wire (`omitempty`), and Harper treats a missing bound as unbounded and scans the whole analytics table (grafana-datasource#177).

Enforced in `datasource.go`: `timeBound.UnmarshalJSON` maps the four exact macro spellings to their named end of the range and `null`/`""` to unset, and rejects anything that is not a positive whole number of milliseconds; `resolve` fills an unset bound from `backend.DataQuery.TimeRange`; `query()` rejects a reversed pair or a non-positive fallback before calling Harper. Named tests: `TestGetAnalyticsTimeBoundsSentToHarper` (the bounds actually sent, through the real SDK client), `TestGetAnalyticsRejectsUnusableTimeBounds` (no request leaves) and `TestQueryDataIsolatesABadQuery` (one bad RefID does not fail its siblings).

The strictness applies to JSON that reaches the backend unparsed, which is the server-side path. The browser path (`applyTemplateVariables` in `src/datasource.ts`) still runs bounds through `parseInt`, so a misspelt macro there becomes `null` and takes the time range. The query editor no longer writes the macros into new queries; the backend keeps accepting them so dashboards saved before 0.2.1 are untouched.
