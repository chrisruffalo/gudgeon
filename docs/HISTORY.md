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

In late July and early August (2019) a huge review was performed of allocations and many, many many 
unnecessary allocations were removed which kept Gudgeon at a consistent ~24MB resident on
my router while serving less than three queries per second. Prior to this review the memory
would slowly inflate to upwards of 55MB of the course of a week. As a side effect of these changes
the engine-reload (when the root config file is changed) no longer takes quite as much memory and is
a much more reasonable option than it used to be.

Also in this same timeframe Gudgeon served it's six millionth query on my network.