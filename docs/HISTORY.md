# History

Gudgeon has been serving **all** of my home DNS traffic since 01/14/2019. Eating my own dogfood has been challenging and there've been a few issues where I've had to get to my laptop and start coding to restore service or bring pihole back online. The intervals between incidents have been steadily increasing. In early March 2019 there was an issue with the reverse lookup feature (specifically Netbios lookup) was causing everything to back up which was eventually fixed.

In late March I almost hit a lifetime queries served of 1,000,000 but I did some major refactoring of the way that both metrics and queries are logged/recorded. This became v0.6.0, which had a few breaking bugs, and eventually became v0.6.1 which is the first build that really achieved both feature parity and "good enough" performance on the Raspberry Pi (first generation) that I tested it on.

