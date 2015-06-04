#Kala

Kala is a simplistic, modern, and performant job scheduler written in Go. It lives in a single binary and does not have any dependencies.

Kala was inspired by Chronos, developed by Airbnb, but the need for a Chronos for the rest of us. Chronos is built on top of Mesos, and
is fault tolerant and distributed by design. These are two features which Kala does not have, as it was built for smaller deployments.

It has a simple JSON over HTTP API, so it is language agnostic. It has a Web UI, Job Stats, Configurable Retries, uses ISO 8601 Date and Interval
notation, Dependant Jobs, and is Persistant (using BoltDB).

### I need [fault tolerance, distributed-features, this to work at scale]

I recommend checking out [Chronos](https://github.com/airbnb/chronos). This is designed to be the Chronos for start-ups.

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
|Getting app-level metrics | GET | /api/v1/stats |


# Documentation

TODO

# Contributing

These are the instructions to follow for working on Kala.

Python is used for some Kala integration tests.


# TODO's

### Features
- [ ] Web UI
- [ ] Config file and/or CL flags
    - Port & Host (needs improvement)
    - Verbose/Debug logging
    - Default owner

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
