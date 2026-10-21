# Design notes

One line per note; the note lives in the directory that owns the code.

- [pkg/plugin](pkg/plugin/DESIGN.md) — every `get_analytics` request the plugin sends carries `0 < start_time <= end_time`, taken from the query or else from Grafana's time range; the backend owns the bound because server-side evaluation never interpolates `${__from}`/`${__to}`.
