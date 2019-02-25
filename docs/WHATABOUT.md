# What About...

## Dnsmasq and Pi-Hole?

Many people reading this are going to point to Pi-Hole or at least Dnsmasq. Those projects are absolutely excellent and they are really inspirations to me as I work on Gudgeon. However, there are a few reasons why these projects are not sufficient for what I am trying to accomplish. I should also note that most people **don't want** what Gudgeon does and, frankly, that's pretty expected. There are also people who need the extra features provided by Pi-Hole or Dnsmasq and that's fine too.

The first reason is what always comes up with Open Source Software: I wanted to do it myself. I have some small experience with DNS manipulation in Go and I [really enjoyed it](https://github.com/chrisruffalo/gyip) so I wanted to do something with a little more complexity.

The other reason is that neither of those solutions will have the right feature set for me without a significant amount of tweaking. A lot of sources say to either use firewall rules and run two instances of your DNS server or to configure each potential client individually. I feel like this is the point I came to and decided to actually do something about it because I really didn't want to run two DNS servers or manage configuration that way... so I spent hours writing a DNS proxy instead.

Pi-hole and Dnsmasq also provide a lot more **DNS** features than Gudgeon ever will. Gudgeon aims to focus on the classification of consumers and what to allow, block, or redirect based on that classification. It does not aim to provide a completely comprehensive DNS proxy or more than a small subset of DNS features.

Finally I wanted to build something that is a little more self-contained and easier to deploy. Gudgeon is a small container-based solution or a single deployable binary with minimal configuration required.