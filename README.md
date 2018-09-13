# Gudgeon

## Overview

Gudgeon is a caching/blocking DNS proxy server. What sets Gudgeon appart is the ability to segregate machines, subnets, and IP ranges into different groups that all receive different blocking rules. The motivation for Gudgeon comes from the proliferation of devices on my home network that belong either to outside entities (Google, AT&T, Amazon), kids, or unwise adults. Different groups need different blocking rules.

Take, for example, a user who has shown persistent inability to avoid internet scams. You can assign that user's machine(s) to group(s) that block more suspicious DNS requests. On the other hand you might want to allow a device like a Google Home or Alexa unit to have full access to the internet except for tracking/advert websites. You might want to create extensive blocklists to protect kids who use the internet from their devices.

For all of these reasons Gudgeon has been created to allow more flexibility in host-based DNS blocking.

## Dnsmasq and Pi-Hole

Many people reading this are going to point to Pi-Hole or at least Dnsmasq. Those projects are absolutely excellent and they are really inspirations to me as I work on Gudgeon. However, there are a few reasons why these projects are not sufficient for what I am trying to accomplish. I should also note that most people **don't want** what Gudgeon does and, frankly, that's pretty expected.

The first is always the reason that comes up with Open Source Software: I wanted to do it myself. I have some small experience with DNS manipulation in Go and I [really enjoyed it](https://github.com/chrisruffalo/gyip) so I wanted to do something with a little more complexity.

The other reason is that neither of those solutions will have the right feature set for me without a significant amount of tweaking. They also provide a lot more *DNS* features than Gudgeon ever will. Gudgeon aims to focus on the classification of consumers and what to allow, block, or redirect based on that classification. It does not aim to provide a completely comprehensive DNS proxy.

Finally I want something that is a little more self-contained and easier to deploy. In the end I expect Gudgeon to be a small container-based solution or a single deployable binary with minimal configuration required.