# Configuration

## Overview
The Gudgeon configuration file serves two main purposes. The first is to define the basic mechanisms for the service (ports, directories, storage) and the 
second is to serve the consumer -> group -> lists -> resolver chain. Understanding the chain of events for a connection request is the cornerstone for creating the
configuration.

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
The home directory is where gudgeon will store all of it's data. In `{home}/cache` downloaded lists are stored, in `{home}/data` you can find any persistent data (like the query log or metrics db), and in `{home}/session` you can find anything that is stored for the term of a single 'engine' session.

```yaml
  network:
    interfaces:
    - ip: 0.0.0.0
      port: 5354
```
Opening network interfaces in this section applies only to the interfaces that will be used for DNS communication. This example sets up port 5354 to listen on all interfaces (0.0.0.0) for both TCP and UDP.