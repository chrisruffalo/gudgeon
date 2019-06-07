# History

Gudgeon has been serving **all** of my home DNS traffic since 01/14/2019. Eating my own dogfood 
has been challenging and there've been a few issues where I've had to get to my laptop and 
start coding to restore service or bring pihole back online. The intervals between incidents 
have been steadily increasing. In early March 2019 there was an issue with the reverse lookup 
feature (specifically Netbios lookup) was causing everything to back up which was eventually 
fixed.

In late March (2019) I almost hit a lifetime queries served of 1,000,000 but I did some major refactoring of the way that both 
metrics and queries are logged/recorded. This became v0.6.0, which had a few breaking bugs, and eventually became v0.6.1 
which is the first build that really achieved both feature parity and "good enough" performance on 
the Raspberry Pi (first generation) that I tested it on.

In April (2019) Gudgeon passed two major milestones on my home network: it served over 1,000,000 
lifetime records and eventually over 1,000,000 records in a single "session" (without being 
restarted). For me this was pretty major because it meant that Gudgeon was capable of being
at least fairly reliable long-term with no evidence of memory leaks over a nearly 30 day
span of handling normal traffic.

In June (2019) Gudgeon gained the ability to reload on source file changes. This feature allows reduced
downtime when updating sources and serves as a prototype of other, similar, reload mechanisms.

