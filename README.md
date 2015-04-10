#Kala

Kala is a simplistic, modern, and performant job scheduler written in Go. It lives in a single binary and does not have any dependencies.

Kala was inspired by Chronos, developed by Airbnb, but the need for a Chronos for the rest of us. Chronos is built on top of Mesos, and
is fault tolerant and distributed by design. These are two features which Kala does not have, as it was built for smaller deployments.

It has a simple JSON over HTTP API, so it is language agnostic. It has a Web UI, Job Stats, Configurable Retries, uses ISO 8601 Date and Interval
notation, Dependant Jobs, and is Persistant (using BoltDB).

# Getting Started

TODO

# API

| Task | Method | Route |
| --- | --- | --- |
|Creating a Job | POST | /api/v1/jobs |
|Getting a Job | GET | /api/v1/jobs/{id} |
|Deleting a Job | DELETE | /api/v1/jobs/{id} |
|Getting a list of all Jobs | GET | /api/v1/jobs |
|Starting a Job manually | POST | /api/v1/jobs/start/{id} |
|Getting app|level metrics | GET | /api/v1/stats |


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
- [ ] Ability to pass in environment vars
- [ ] Config file and/or CL flags
    - Port & Host

### For User
- [ ] Users Documentation
- [ ] Single command to run docker image
- [ ] Python Client Library
- [ ] Node Client Library
- [ ] CLI
- [ ] Create single release binary

### For Contributors
- [ ] Contributors Documentation
- [ ] Continious Integration

