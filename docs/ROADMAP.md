# Features Roadmap

In no particular order these are the future improvements that are coming to Gudgeon:

* General
  * Ability to stop/start/restart different consumers (provider, web)
  * Integration tests that use start/stop mechanics for end-to-end testing
  * File watching and reloading minimal configuration changes without service restart or downtime (hot reload)
* Metrics
  * **Done:** Local instance metrics storage
  * Exporting to Prometheus and InfluxDB (followed by others as desired)
* Query Log
  * Reverse name lookups for clients (possibly netbios names or other names if possible for things not in DNS infrastructure)
* Web UI
  * Searchable query log
  * Metrics graph widgets
* Logging:
  * Use actual, configurable, logging framework instead of fmt.Printf
* Rules
  * **Done:** SQLite3 Storage engine (will require even lower memory but requires disk space/access and will slow down resolution some)
  * **Done:** Bloom-filter-based storage engine that can use SQLite3 to guard against false positives
  * Refactor code in rule storage to use common code path for querying and delegation to backing store (easily enable "hash+sqlite", "hash32+sqlite")
* Configuration
  * **Done:** Better default configuration (more clear on what omitted values mean)
  * **Done:** Warnings for configuration elements that might cause issues
  * Ability to write default/simple configuration as command line option
  * "conf.d"-like capability to merge/include multiple configuration files
  * **Done:** Configuration checking/parsing with warnings and errors (from command line too)
* Resolution
  * **Done:** Built-in "system" source/resolver to use OS's resolution (through Go API)
  * Configurable "system" resolver for resolving domain names internally
  * Conditional resolution (only use certain resolvers in certain conditions)
  * Using resolv.conf files as resolution sources
  * Name support with DNS-Over-TLS (use domain name instead of just IP as resolver source)
* Consumers
  * **Done:** Block clients at the consumer level
  * Invert consumer matching (or more sophisticated consumer matching)
* Groups
  * "Inherit" from other groups (heirarchy of groups)
* DNS Features
  * DNSSEC checking support 
  * DNSSEC signature support
  * DNS-Over-HTTP support (client, server)
  * DNS-Over-TLS support (server)

