module github.com/ajvb/kala/client/v2

go 1.26

// Until the client has its own types (a later change) it still imports server
// packages, so it points at the server module in this repository. A replace
// is only honoured in the main module, so this module is not usable by other
// modules yet.
require github.com/ajvb/kala/v2 v2.0.0-00010101000000-000000000000

require (
	github.com/gorilla/mux v0.0.0-20150808061613-ffb3f683aad4
	github.com/stretchr/testify v1.7.0
)

require (
	github.com/cornelk/hashmap v1.0.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dchest/siphash v1.1.0 // indirect
	github.com/elazarl/go-bindata-assetfs v1.0.1 // indirect
	github.com/gorilla/context v0.0.0-20141217160251-215affda49ad // indirect
	github.com/mattn/go-shellwords v1.0.0 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/phyber/negroni-gzip v0.0.0-20180113114010-ef6356a5d029 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sirupsen/logrus v1.8.3 // indirect
	github.com/urfave/negroni v1.0.0 // indirect
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

replace github.com/ajvb/kala/v2 => ../
