# bcast - simple http-based broadcast
This package implements a simple broadcasting service that is soley build on
the http protocol. Befor deciding on this direction we looked at various
alternatives

  - WebRTC - too complicated and requires central servers for signaling and
    NAT traversal and is overall quite complicated. It also still requires
    peer discovery to be solved separately.
  - UPnP - this and, other router protocols, for nat traversal are hit and miss
    by relying on admins to have routers configured with this option enabled.
  - BitCoin - simpler gossip and proven in for this usecase but sends traffic
    over its own port which makes it hard to use in corporate environments.

Instead we accepted the a design with the following properties and requirements.

  - We accept some star topology. Some users will always be participating on
    deviced that cannot directly be reached from the outside.
  - Some corporation will have proxies in place to mitm HTTPS traffic, even
    in such environments should traffic be private.
  - In public WiFi spots only HTTP(s) traffic is allowed anyway. So if we
    want to make it usable for these users we need http.  
  - The implementation can be made way simpler

Simply put, HTTP is largest common demoninator when it comes to making the
broadcast usable for the widest audience. Besides this there are other benefits:

  - The Go standard library has excellent support for http services
  - Large nodes can apply all the standard practices for handling large loads
  - A web ui might be exposed that allows submitting transactions through the browser

## TODO
- [x] write a basic storage file that can configure peers
- [ ] peers with a port other then 80 should be selectable without a explicit flag
- [ ] can we select peers on last rtt time (lower is better)
- [ ] make it possible to efficiently ask several peers for a specific block (or tree)
- [ ] make it possible to fan-out a message to several peers
- [ ]
