# Gudgeon [![Build Status](https://travis-ci.org/chrisruffalo/gudgeon.svg?branch=master)](https://travis-ci.org/chrisruffalo/gudgeon) [![Go Report Card](https://goreportcard.com/badge/github.com/chrisruffalo/gudgeon)](https://goreportcard.com/report/github.com/chrisruffalo/gudgeon)

## Overview

Gudgeon is a caching/blocking DNS proxy server. What sets Gudgeon appart is the ability to segregate machines, subnets, and IP ranges into different groups that all receive different blocking rules. The motivation for Gudgeon comes from the proliferation of devices on my home network that belong either to outside entities (Google, AT&T, Amazon), kids, or unwise adults. Different groups need different blocking rules.

Take, for example, a user who has shown persistent inability to avoid internet scams. You can assign that user's machine(s) to group(s) that block more suspicious DNS requests. On the other hand you might want to allow a device like a Google Home or Alexa unit to have full access to the internet except for tracking/advert websites. You might want to create extensive blocklists to protect kids who use the internet from their devices.

For all of these reasons Gudgeon has been created to allow more flexibility in host-based DNS blocking.

## Features

* Go Routines for non-blocking request handling enables high-througput especially with simultaneous requests
* Systemd Integration to run as non-root user (with access to priveleged ports through Systemd sockets)
* Configure upstream DNS types (tcp-tls/dns-over-tls, tcp, and udp) explicitly
* Using regular expressions and wildcards to block DNS names
* Matching an address (or subnet, or subnet range) to a user and determining what blocklists to use
* Having resolvers for certain/specific subnets based on matching incoming connections
* Inline host file entries in configuration file
* Enhanced (and backwards-compatible) hostname format supports wildcard names, CNAME/PTR entries, and reverse lookups

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

## Resolvers

In Gudgeon a "resolver" is a set of configuration and sources that are used to resolve DNS queries. Each resolver is a description of several aspects of name resolution.

## Configuration and Examples

**TODO**

## What About Dnsmasq and Pi-Hole?

Many people reading this are going to point to Pi-Hole or at least Dnsmasq. Those projects are absolutely excellent and they are really inspirations to me as I work on Gudgeon. However, there are a few reasons why these projects are not sufficient for what I am trying to accomplish. I should also note that most people **don't want** what Gudgeon does and, frankly, that's pretty expected. There are also people who need the extra features provided by Pi-Hole or Dnsmasq and that's fine too.

The first reason is what always comes up with Open Source Software: I wanted to do it myself. I have some small experience with DNS manipulation in Go and I [really enjoyed it](https://github.com/chrisruffalo/gyip) so I wanted to do something with a little more complexity.

The other reason is that neither of those solutions will have the right feature set for me without a significant amount of tweaking. A lot of sources say to either use firewall rules and run two instances of your DNS server or to configure each potential client individually. I feel like this is the point I came to and decided to actually do something about it because I really didn't want to run two DNS servers or manage configuration that way... so I spent hours writing a DNS proxy instead.

Pi-hole and Dnsmasq also provide a lot more **DNS** features than Gudgeon ever will. Gudgeon aims to focus on the classification of consumers and what to allow, block, or redirect based on that classification. It does not aim to provide a completely comprehensive DNS proxy or more than a small subset of DNS features.

Finally I wanted to build something that is a little more self-contained and easier to deploy. Gudgeon is a small container-based solution or a single deployable binary with minimal configuration required.

## Building
Prerequisites
* Ability to use Makefiles (`make` command installed)
* Go > 1.11 (module support is *required*)
* `upx` (for binary compression)
* `fpm` (for building deb/rpm)
* Docker (for building docker images)

With the prerequisites installed you can build Gudgeon by...
* Downloading vendor assets (patternfly, vue, etc) with `[]$ make download`
* Preparing your environment with needed Go tools with `[]$ make prepare`
* Building the binary with `[]make`

The `download` target is used to download new dependencies when needed. The `prepare` target is only needed if the required Go tools change. The output of the process is a statically compiled for a few different platforms. The binary is statically compiled to make it easily portable to platforms and other systems that do not have Golang compilers.