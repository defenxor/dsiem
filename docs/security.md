# Notes on Security

This section briefly describes how Dsiem design and implementation addresses security concerns.

*If you instead want to report security-sensitive information (like a potential security bug), then please send an email to devs@defenxor.com. The GPG public key for that address can be found [here](https://pgp.mit.edu/pks/lookup?search=devs%40defenxor.com)*.

On the design side, we aimed to:
* Eliminate attack surface by only implementing the bare minimum functionality, and relying on other infrastructure components to do the rest. For instance, there's no authentication on the web interface because Nginx or other similar frontends can easily be used to provide that with more options and manageability (we personally use TLS with client certificates).
* Adopt least-privilege principle. Dsiem binary requires no special privileges, and only needs to have read access to its own directory, and write access to logs and configs subdirectories.
* Provide secure defaults. For instance, Write access to configs directory is only needed by web UI and is therefore turned off by-default.

On the implementation side, we tried to:
* Check and handle all errors appropriately. Go verbose error handling style and early return convention makes it easy to reason about errors and their potential impact. 
* For the HTTP endpoint part, obviously we try to check all user inputs and return [418 status code](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/418) as needed.
