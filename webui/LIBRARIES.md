# Web interface libraries

The web interface uses these third-party libraries. They are not stored in
the repository: `make webui-libs` (part of `make bin/kala` and of the
container build) installs them with npm from `package.json` and
`package-lock.json`, which pins each version and its integrity hash, and
copies the files the interface loads into `vendor/`, where they are embedded
in the binary. Building the web interface from source therefore needs npm. A
binary built without the libraries, for example with a plain `go build`,
serves a page saying the web interface is not available; the API works as
usual.

| Library | Package | Version | License |
| --- | --- | --- | --- |
| [Font Awesome Free](https://github.com/FortAwesome/Font-Awesome) | `@fortawesome/fontawesome-free` | 5.3.1 | Icons CC BY 4.0, fonts SIL OFL 1.1, code MIT |
| [Bulma](https://github.com/jgthms/bulma) | `bulma` | 0.8.2 | MIT |
| [Reef](https://github.com/cferdinandi/reef) | `reefjs` | 7.3.0 | MIT |

GitHub's dependency graph reads `package-lock.json`, so alerts are raised for
these packages.

## Reviewing and updating them

At every release, and when GitHub raises an alert, check each package and
version above for known vulnerabilities (`npm audit` in this directory, or
[osv.dev](https://osv.dev/)). Last checked 2026-11-22: no known
vulnerabilities.

To update a library, run `npm install --save-exact <package>@<version>` in
this directory, update this table, and check the files that the
`webui-libs` target in the Makefile copies still exist.
