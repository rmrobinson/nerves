# nanoleafd

This service allows a single Nanoleaf device to be controlled using the domotics framework. The UUID of the device, plus the API key, are configured at initialization; the service then monitors SSDP broadcasts to find the Nanoleaf API, connects and begins translating commands.

It currently does not subscribe to updates from the Nanoleaf API due to poor performance noticed; it is suggested this is the only thing which controls the Nanoleaf entity for consistency.
