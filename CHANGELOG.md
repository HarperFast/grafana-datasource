# Changelog

## 0.2.0

- Added an **"Aggregate analytics across cluster nodes"** data source setting (off by default). When enabled, the plugin requests cluster-wide analytics from the single node it targets: that node fans the `get_analytics` query out to its peers and merges the results, so one data source shows series for every node (split by the `node` label) even when analytics replication is disabled. **Requires Harper 5.1.0 or later** for cross-node aggregation (the release that added `get_analytics` peer fan-out); older Harper ignores the option and returns only the queried node's analytics.
- Removed the `@grafana/plugin-ui` dependency (an internal-only package not permitted for catalog-published plugins); the config editor now uses `@grafana/ui` primitives.
- Upgraded React to 19 and confirmed the plugin runs on Grafana 13.x.

## 0.1.6

- Fixed plugin failing to load on Grafana 13+ (React 19) with `Cannot read properties of undefined (reading 'ReactCurrentOwner')`. The React JSX runtime is now provided by Grafana instead of being bundled with the plugin.
- Minimum Grafana version is now 11.6.11 / 12.0.10 / 12.1.7 / 12.2.5 (the versions that provide the shared JSX runtime).

## 0.1.5

- Spiffied up the config form a bit
- Added support skipping TLS verification in the Harper connection
- Started passing the `coalesce_time=true` param to Harper's `get_analytics` API to get connected line graphs from replicated analytics

## 0.1.4

- Make datasource a variable in example dashboard so it can actually be used 😂

## 0.1.3

- Added example dashboard

## 0.1.2

- Added ability to trace HTTP requests for better debugging plugin issues.
- Updated dependency versions to latest as of release.

## 0.1.1

- Fixed a performance issue loading metrics from Harper. Note that the fix requires Harper 4.6.19+ or 4.7.8+.
- Updated dependency versions to latest as of release.

## 0.1.0

Initial release.
