## Practical Example Configuration
This configuration is a practical example that I use on my home router to manage my traffic. I've included comments in the example to explain why certain things are being done.

```yaml
gudgeon:
  # this is the default value that the rpm package uses
  home: /var/lib/gudgeon/

  # use the bloom filter backed by sqlite to get a low memory footprint
  # with the speed and density of the bloom filter. good balance of performance
  # and memory
  storage:
    rules: "bloom+sqlite"

  # queries don't need to go to stdout for my use
  query_log:
    stdout: false

  resolvers:
  # for a lot of services it seems that my isp has either restrictions
  # or other things in their network that just need to be done a certain
  # way and they fail with public DNS resolution 
  - name: isp-upstream
    domains:
    - edgesuite.net
    - quickplay.com
    - apptentive.com
    - espn.com
    - cloudfront.net
    - go.com
    - edgekey.net
    - apple.com
    sources:
    - 192.168.1.254 # this is the closest upstream dns provided by the ISP

  # my local network has several services that need domain names
  # typically these have the ".lan" suffix and the search domain here
  # provides that if the query needs it
  - name: local
    search:
    - lan
    sources:
    - /etc/gudgeon/gudgeon-hosts

  # the hostfile format for gudgeon can provide aliases. in this case
  # we use these aliases to ensure that google.com and bing.com both
  # use the "safe" version of their searches
  - name: safe
    hosts:
    - forcesafesearch.google.com google.com www.google.com
    - strict.bing.com www.bing.com

  # all non-lan domains (that aren't matched already) are resolved by google 
  - name: google
    skip:
    - "*.lan"
    sources:
    - 8.8.8.8/tcp-tls
    - 8.8.4.4/tcp-tls
    - 8.8.8.8
    - 8.8.4.4

  # the default resolver lays out the order to use the other resolvers
  # in this case local resolution is tried, then isp resolution, and 
  # finally google
  - name: default
    sources:
    - local
    - isp-upstream
    - google

  lists:
  # i used to use pihole and so referencing the whitelist and blacklist this way 
  # means that gudgeon can be a drop-in replacement
  - name: global white list
    src: /etc/pihole/whitelist.txt
    type: allow
  - name: global black list
    src: /etc/pihole/blacklist.txt
  # generic list that comes with pihole by default, or at least used to, and 
  # mostly works for my application
  - name: stephen's black list
    src: https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts
  - name: cameleon
    src: http://sysctl.org/cameleon/hosts
    src: https://zeustracker.abuse.ch/blocklist.php?download=domainblocklist
  - name: disconnect.me tracking
    src: https://s3.amazonaws.com/lists.disconnect.me/simple_tracking.txt
  - name: disconnect.me ads
    src: https://s3.amazonaws.com/lists.disconnect.me/simple_ad.txt
  # an additional block list that is applied to machines on the network that belong
  # to parties that need additional protection/blocking    
  - name: chad's adult block
    src: https://raw.githubusercontent.com/chadmayfield/my-pihole-blocklists/master/lists/pi_blocklist_porn_top1m.list
    tags:
    - safe

  groups:
  # the default group uses the default resolution chain
  - name: default
    tags:
    - default # this is actually implicit
    resolvers:
    - default

  # the safe group uses the safe resolution chain and then
  # the default resolution chain once the safe chain is complete
  - name: safe
    tags:
    - default
    - safe # it also uses the "safe" tag to apply the "adult block" list
    resolvers:
    - safe
    - default

  consumers:
  # the default consumer matches all traffic that doesn't match any other consumer
  - name: default
    groups:
    - default # and uses the default group to resolve/block traffic
  
  # the safe consumer matches specific IPs that belong to devices that need
  # more blocking/filtering
  - name: safe
    groups:
    - safe # and uses the safe group to resolve/block traffic
    matches:
    - ip: 192.168.0.179 
    - ip: 192.168.0.221
```