# Installed libraries

`make webui-libs` (part of `make bin/kala`) installs the web interface's
third-party libraries with npm and copies the files it loads here. Only this
file is committed. A binary built without them serves a page saying the web
interface is not available.
