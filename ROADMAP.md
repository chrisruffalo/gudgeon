# Features Roadmap

In no particular order these are the future improvements that are coming to Gudgeon:

* Metrics
  * Local instance metrics storage
  * Exporting to Prometheus and InfluxDB (followed by others as desired)
* Web UI
  * Searchable query log
  * Metrics widgets
* Configuration
  * Better default configuration (more clear on what omitted values mean)
  * Warnings for configuration elements that might cause issues
  * Ability to write default/simple configuration as command line option
  * "conf.d"-like capability to merge/include multiple configuration files
  * Configuration checking/parsing with warnings and errors (from command line too)
* Resolution
  * Configurable "system" resolver for resolving domain names internally
  * Conditional resolution (only use certain resolvers in certain conditions)
  * Using resolv.conf files as resolution sources
  * Name support with DNS-Over-TLS (use domain name instead of just IP as resolver source)
* Consumers
  * Block clients at the consumer level
  * Invert consumer matching
* Groups
  * "Inherit" from other groups (heirarchy of groups)
* DNS Features
  * DNSSEC checking support 
  * DNSSEC signature support
  * DNS-Over-HTTP support (client, server)
  * DNS-Over-TLS support (server)

