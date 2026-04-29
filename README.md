# goutil

A set of utility packages for Go:

* `erru`
  * `Must` helper for panic-on-error value extraction.
* `logu`
  * Context-scoped log metadata helpers (`ExtendLogContext`) to propagate fields like request/user IDs through derived contexts.
  * `PlainLogHandler` for compact, grep-friendly text logs with timestamp, level, message, context metadata, and slog attrs.
  * Cross-platform default log directory resolution (`ResolveLogDir`) for macOS and Linux.
