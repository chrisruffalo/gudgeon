# Configuration

## Overview
The Gudgeon configuration file serves two main purposes. The first is to define the basic mechanisms for the service (ports, directories, storage) and the 
second is to serve the consumer -> group -> lists -> resolver chain. Understanding the chain of events for a connection request is the cornerstone for creating the
configuration.

## Common Rules
In Gudgeon there are a few common rules for all the different types of configuration:
* If an element is named that name **must** be unique within the list of all the other elements of the same type. You cannot have two resolvers named "google" but
  having a group *and* resolver named "google" is fine.
* Elements are processed in order from the top to the bottom. Resolvers, consumers, groups all will be matched/processed/evaluated in that order.

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
  home: /opt/gudgeon
```
The home directory is where gudgeon will store all of its data. In `{home}/cache` downloaded lists are stored, in `{home}/data` you can find any persistent data (like the query log or metrics db), and in `{home}/session` you can find anything that is stored for the term of a single 'engine' session.

```yaml
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
    - name: 'google'
      sources:
      - 8.8.8.8
      - 8.8.8.4
```

This resolver defines two upstream sources named 'google'. Each source will be tried **in order** until a non-empty response is found.

## Groups


## Consumers
