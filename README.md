# Gudgeon

## Overview

Gudgeon is a caching/blocking DNS proxy server. What sets Gudgeon appart is the ability to segregate machines, subnets, and IP ranges into different groups that all receive different blocking rules. The motivation for Gudgeon comes from the proliferation of devices on my home network that belong either to outside entities (Google, AT&T, Amazon), kids, or unwise adults. Different groups need different blocking rules.

Take, for example, a user who has shown persistent inability to avoid internet scams. You can assign that user's machine(s) to group(s) that block more suspicious DNS requests. On the other hand you might want to allow a device like a Google Home or Alexa unit to have full access to the internet except for tracking/advert websites. You might want to create extensive blocklists to protect kids who use the internet from their devices.

For all of these reasons Gudgeon has been created to allow more flexibility in host-based DNS blocking.

## Concept of Operations

Gudgeon matches incoming traffic to a consumer, maps the consumer to a set of groups, each group supports multiple different lists, and then finally to resolvers. Each step is designed to provide flexibility to end users.

When a request is received Gudgeon does the following:
* Check for a matching consumer, if no consumer is matched, use the "default" consumer.
* Check for matching groups, if no groups are matched, use the "default" group.
* Determine if the domain is blocked based on the lists that belong to the matched group(s)
* Get resolvers for groups, if no resolvers are matched,  use the "default" resolver.
* Attempt to resolve the domain using the resolvers that belonged to the groups
* Return the result

The point is to provide several different ways to change behavior away from the standard (and single-pathed) resolution heirarchy that is familiar to us from most DNS providers. The main way to do this is by assigning consumers to different groups (subnets, IPs, IP ranges) or by structuring resolvers to work in a way that more accurately reflects the needs of your site.

## Resolvers

In Gudgeon a "resolver" is a set of configuration and sources that are used to resolve DNS queries. Each resolver is a description of several aspects of name resolution.

## Configuration and Examples

**TODO**

## Comparison to Dnsmasq and Pi-Hole

Many people reading this are going to point to Pi-Hole or at least Dnsmasq. Those projects are absolutely excellent and they are really inspirations to me as I work on Gudgeon. However, there are a few reasons why these projects are not sufficient for what I am trying to accomplish. I should also note that most people **don't want** what Gudgeon does and, frankly, that's pretty expected.

The first is always the reason that comes up with Open Source Software: I wanted to do it myself. I have some small experience with DNS manipulation in Go and I [really enjoyed it](https://github.com/chrisruffalo/gyip) so I wanted to do something with a little more complexity.

The other reason is that neither of those solutions will have the right feature set for me without a significant amount of tweaking. They also provide a lot more *DNS* features than Gudgeon ever will. Gudgeon aims to focus on the classification of consumers and what to allow, block, or redirect based on that classification. It does not aim to provide a completely comprehensive DNS proxy.

Finally I wanted to build something that is a little more self-contained and easier to deploy. In the end I expect Gudgeon to be a small container-based solution or a single deployable binary with minimal configuration required.

## Building
Prerequisites
* Go > 1.11 (module support is *required*)

With the prerequisites you need to do to build Gudgeon is `[user@host]$ make` and the output binary will be `build/gudgeon` statically compiled for the platform you built it on. The binary is statically compiled to make it easily portable to platforms and other systems that do not have Golang compilers.