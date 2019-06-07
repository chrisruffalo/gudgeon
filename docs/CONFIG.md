# Configuration

## Overview
The Gudgeon configuration file serves two main purposes. The first is to define the basic mechanisms for the service (ports, directories, storage) and the 
second is to serve the consumer -> group -> lists -> resolver chain. Understanding the chain of events for a connection request is the cornerstone for creating the
configuration.

## Practical Advice
For the vast majority of users a simple example configuration will work best. For a practical example based on my personal home configuration [see here](/docs/PRACTICAL.md).

## Common Rules
In Gudgeon there are a few common rules for all the different types of configuration:
* If an element is named that name **must** be unique within the list of all the other elements of the same type. You cannot have two resolvers named "google" but
  having a group *and* resolver named "google" is fine.
* In contradiction to the above: a Source and Resolver should not have the same name.
* Elements are processed in order from the top to the bottom. Resolvers, Sources, Consumers, and Groups all will be matched/processed/evaluated in that order.

## Basic Configuration
In this section we take a look at a simple configuration file and show how it can be changed or expanded to match your use case. This configuration is very basic.

```yaml
gudgeon:
  home: /opt/gudgeon

  network:
    interfaces:
    - ip: 0.0.0.0
      port: 5354

  resolvers:
  - name: default
    sources:
    - 8.8.8.8
    - 8.8.4.4

  lists:
  - name: stephen's black list
    src: https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts
```
You can note first that no consumers or groups are defined. That is because, by default, there is a "default" consumer and "default" group. The default consumer points to the default group and the default group points to the default resolver and any lists tagged with default.
```yaml
gudgeon:
  lists:
  - name: stephen's black list
    src: https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts
    tags:
    - default

  groups:
  - name: default
    resolvers:
    - default
    tags:
    - default

  consumer:
  - name: default
    groups:
    - default
```
This is what the configuration looks like with the default group and the default consumer. The default consumer and resolver can be overriden or changed but for most purposes they should be left alone.

```yaml
gudgeon:
  home: /opt/gudgeon
```
The home directory is where gudgeon will store all of its data. In `{home}/cache` downloaded lists are stored, in `{home}/data` you can find any persistent data (like the query log or metrics db), and in `{home}/session` you can find anything that is stored for the term of a single 'engine' session.

```yaml
gudgeon:
  network:
    interfaces:
    - ip: 0.0.0.0
      port: 5354
```
Opening network interfaces in this section applies only to the interfaces that will be used for DNS communication. This example sets up port 5354 to listen on all interfaces (0.0.0.0) for both TCP and UDP.

## Resolvers

A Gudgeon resolver is a configuration item that groups together different DNS sources. These can be a flat host file, an upstream dns server, a zone db file, or another resolver. Each resolver, like many other Gudgeon elements, must have a unique name.

Take a look at this simple resolver:

```yaml
gudgeon:
  resolvers:
    - name: 'google'
      sources:
      - 8.8.8.8
      - 8.8.8.4
```

This resolver defines two upstream sources named 'google'. Each source will be tried **in order** until a non-empty response is found.

Resolvers can be configured for specific domains. You might use this to ensure certain ISP-related features only use the DNS of the ISP or that VPN-based requests use the correct upstream DNS.
```yaml
gudgeon:  
  resolvers:
  - name: "att"
    domains:
    - "att.net"
    - "*apple.com"
    sources:
    - < upstream AT&T dns >
    - < upstream AT&T dns >
```
In this example if a domain matches or ends with *the label* "att.net" then it will be sent to the defined upstream sources. The domains "mail.att.net" and "att.net" match but the domain "goatt.net" would not because the label "att" doesn't match. This domain property also supports wildcards int he form of "*". The example domain match "*apple.com" would match **any** domain that ends with "apple.com" so "getapple.com", "apple.com", and "cloud.apple.com" would all match. Queries for domains that do not match will not be served by this Resolver.   

In a similar way a Resolver can skip queries made for a given domain. For example you may not want a resolver to resolve certain domains and instead defer that to another source. This is similar to the domain match feature but it allows more flexibility in organizing resolver order. Skip domains obey the same rules for matching as domains.
```yaml
gudgeon:
  resolvers:
  - name: "local"
    skip: 
    - ".com"
    sources:
    - /var/lib/data/local.zone.db
  - name: "upstream"
    sources:
    - 192.168.1.254  
```
If the resolvers were used in order ("local" and then "upstream") any ".com" domains would be passed over. 

## Sources
A source is any mechanism that a resolver can use to resolve a DNS query. Gudgeon supports the following sources:
* Upstream DNS by IP
* Local file resolution (hostfile, zone db file, resolv.conf)
* Fallback to the system resolver

Sources can be configured in two places. The simplest way is as a source that belongs to a resolver.
```yaml
gudgeon:
  resolvers:
    - name: 'everything'
      sources:
      - /etc/resolv.conf
      - /tmp/hostfile
      - /var/lib/dns/home.local.db
      - 8.8.8.8/tcp-tls
      - 8.8.4.4/tcp
```
This creates a single resolver that uses the sources listed, in order, to resolve DNS names. This would check the resolve file, the hostfile, the local zonedb, and then the remote sources at 8.8.8.8 and 8.8.4.4.

Another way to create sources is to configure them separately by name:
```yaml
gudgeon:
  sources:
  - name: "google-tls"
    load_balance: true
    spec:
    - "8.8.8.8/tcp-tls"
    - "8.8.4.4/tcp-tls"
  - name: "google"
    - "8.8.8.8/tcp"
    - "8.8.4.4/tcp"
  resolvers:
  - name: "google-resolver"
    sources:
    - google-tls
    - google
```
This example shows two configured sources. The "google-tls" source will balance requests between the two Google tcp-tls endpoints. The "google" source will try each tcp endpoint in order until a response is found. The "google-resolver" given will use the google-tls source and, if no answer is found for the query the next source will be tried. 

It is **very** important to ensure that your sources and resolvers do not share names as they can easily occlude one another leading to incorrect or unpredictable resolution.

## Groups


## Consumers
