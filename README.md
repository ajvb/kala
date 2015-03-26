#Kala

Kala is a simplistic, modern, and performant job scheduler written in Go. It lives in a single binary and does not have any dependencies.

Kala was inspired by Chronos, developed by Airbnb, but the need for a Chronos for the rest of us. Chronos is built on top of Mesos, and
is fault tolerant and distributed by design. These are two features which Kala does not have, as it was built for smaller deployments.

It has a simple JSON over HTTP API, so it is language agnostic. It has a Web UI, Job Stats, Configurable Retries, uses ISO 8601 Date and Interval
notation, Dependant Jobs, and is Persistant (using BoltDB).

# Getting Started

TODO

# Documentation

TODO

# Dev Documentation

These are the instructions to follow for working on Kala.

Python is used for some Kala integration tests.

# License

MIT


# TODO's

### Features
- [ ] Web UI
- [ ] Job Stats (e.g. 50th, 75th, 95th and 99th percentile timing, failure/success)
- [ ] Go Client Library
- [ ] Disaster Recovery
- [ ] Error Reporting
- [ ] RunAsUser
- [ ] Ability to pass in environment vars

#### Performance
- [ ] Swap AllJobs map out with a really caching backend

### For User
- [ ] Users Documentation
- [ ] Getting Started Guide
- [ ] Single command to run docker image
- [ ] More Example Scripts
- [ ] Python Client Library
- [ ] Node Client Library
- [ ] CLI

### For Contributors
- [ ] Contributors Documentation
- [ ] Continious Integration

