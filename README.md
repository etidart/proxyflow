# proxyflow - the proxy relay

_this program periodically checks proxies, measures their response times, and
redirects user's requests to the best proxies_

caveats:

- timeouts and many other stuff are hardcoded (see internal/constants) but do they
  really need to be changed with the config? if so, that is not too hard to
  implement a config system
- there is no functionality for adding/deleting proxies while the program is
  running. IPC methods are too complicated when you can just respawn proxyflow
  with updated pfile

needs to be fixed:

- issues with host's network connection will result in the deletion of all proxies
  (if proxyflow is being used, it will result in rotating bad/normal proxy lists)
