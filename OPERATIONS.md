## Concept of Operations

Gudgeon matches incoming traffic to a consumer, maps the consumer to a set of groups, each group supports multiple different lists, and then finally to resolvers. Each step is designed to provide flexibility to end users.

When a request is received Gudgeon does the following:
* Check for a matching consumer, if no consumer is matched, use the "default" consumer.
* Check for matching groups, if no groups are matched, use the "default" group.
* Determine if the domain is blocked based on the lists that belong to the matched group(s)
* Get resolvers for groups, if no resolvers are matched,  use the "default" resolver.
* Attempt to resolve the domain using the resolver sources that belonged to the groups
* Return the result from the source or the result of a blocked request

The point is to provide several different ways to change behavior away from the standard (and single-pathed) resolution heirarchy that is familiar to us from most DNS providers. The main way to do this is by assigning consumers to different groups (subnets, IPs, IP ranges) or by structuring resolvers to work in a way that more accurately reflects the needs of your site.