# Features Roadmap

In no particular order these are the future improvements that are coming to Gudgeon:

* General
  * **In Progress:** Ability to stop/start/restart different consumers (provider, web)
  * **In Progress:** Integration tests that use start/stop mechanics for end-to-end testing
  * **In Progress:** File watching and reloading minimal configuration changes without service restart or downtime (hot reload)
  * **In Progress:** Package refactoring to put `rule`, `resolver`, `qlog`, and `metrics` in engine package (at a minimum)
  * **Done:** Reverse lookup moved outside of query log and into part of engine
  * Ability to track query latency/processing time
  * **Done:** Unindexed single tables for SQL insert speed before batching into long-term storage
* Metrics
  * **Done:** Local instance metrics storage
  * Better CPU graph
  * Memory graph with more memory classes (system or golang stats)
  * **Done:** Cache size as second graph with memory (??)
  * Exporting to Prometheus and InfluxDB (followed by others as desired)
  * Condensing data in the database based on time interval
* Query Log
  * **Done:** Reverse name lookups for clients (possibly netbios names or other names if possible for things not in DNS infrastructure)
  * **Done:** Zeroconf/mDNS/Avahi/Bonjour compatible lookups for reverse name finding
* Web UI
  * **In Progress: ** Searchable query log
  * **Done:** Metrics graph widgets
  * **Done:** Query Tester (requires engine refactoring)
* Logging:
  * **Done:** Use actual, configurable, logging framework instead of fmt.Printf
  * Implement file logging for main logger
* Rules
  * **Done:** SQLite3 Storage engine (will require even lower memory but requires disk space/access and will slow down resolution some)
  * **Done:** Bloom-filter-based storage engine that can use SQLite3 to guard against false positives
  * **Done** Refactor code in rule storage to optionally sql backing store (easily enable "hash+sqlite", "hash32+sqlite")
* Configuration
  * **Done:** Better default configuration (more clear on what omitted values mean)
  * **Done:** Warnings for configuration elements that might cause issues
  * Ability to write default/simple configuration as command line option
  * "conf.d"-like capability to merge/include multiple configuration files
  * **Done:** Configuration checking/parsing with warnings and errors (from command line too)
* Resolution
  * **Done:** Built-in "system" source/resolver to use OS's resolution (through Go API)
  * **In Progress:** Configurable "system" resolver for resolving domain names internally
  * Conditional resolution (only use certain resolvers in certain conditions)
  * **In Progress:** Using resolv.conf files as resolution sources
  * **Done:** Using Zone-files (\*.db) as a resolution source
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

