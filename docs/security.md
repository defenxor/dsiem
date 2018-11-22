# Security

### Design
- Eliminates attack surface by only implementing the bare minimum functionality, and relies on other infrastructure components to do the rest. For instance, there's no authentication on the web interface because Nginx or other similar frontends can easily be used to provide that with more options and managebility (we personally use TLS with client certificates).
- Implement least-privilege principle, only needs to have read and access to its own directory.

### Implementation
- Check and handle all errors appropriately. Go verbose error handling style and early return convention makes it easy to reason about this.
- Check all user inputs on the HTTP API endpoint, returns 418 status code as needed.
