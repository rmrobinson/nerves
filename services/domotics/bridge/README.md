# bridge

The `bridge` package contains a number of implementations of different domotic bridge implementations. Each bridge may export one or more devices; when a bridge is added to the `domotics.Hub` it becomes accessible to clients of the Hub API.

The bridge implementations may be synchronous or asynchronous; see `domotics.SyncBridge` and `domotics.AsyncBridge` for the relevant interfaces to satisfy.

There are mock implementations of a generic bridge available in the `mock` sub-package for testing purposes.