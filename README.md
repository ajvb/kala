#Kala

Kala is a simplistic, modern, and performant job scheduler written in Go. It lives in a single binary and does not have any dependencies.

Kala was inspired by [Chronos](https://github.com/airbnb/chronos), developed by Airbnb, but the need for a Chronos for the rest of us. Chronos is built on top of Mesos, and
is fault tolerant and distributed by design. These are two features which Kala does not have, as it was built for smaller deployments.

It has a simple JSON over HTTP API, so it is language agnostic. It has a Web UI, Job Stats, Configurable Retries, uses ISO 8601 Date and Interval
notation, Dependant Jobs, and is Persistant (using BoltDB).

#### I need [fault tolerance, distributed-features, this to work at scale]

I recommend checking out [Chronos](https://github.com/airbnb/chronos). This is designed to be the Chronos for start-ups.

# Getting Started

TODO

### Examples of Usage

There are more examples in the examples directory within this repo. Currently its pretty messy. Feel free to submit a new example if you have one.

# API v1 Docs

All routes have a prefix of `/api/v1`

## Client Libraries

#### Official:
* [client](https://github.com/ajvb/kala/tree/master/client) - go

## Overview of routes

| Task | Method | Route |
| --- | --- | --- |
|Creating a Job | POST | /job |
|Getting a list of all Jobs | GET | /job |
|Getting a Job | GET | /job/{id} |
|Deleting a Job | DELETE | /job/{id} |
|Getting metrics about a certain Job | GET | /job/stats/{id} |
|Starting a Job manually | POST | /job/start/{id} |
|Getting app-level metrics | GET | /stats |

## /job

This route accepts both a GET and a POST. Performing a GET request will return a list of all currently running jobs.
Performing a POST (with the correct JSON) will create a new Job.

Example:
```
ajvb@cloak:~/gocode/src/github.com/ajvb/kala$ curl http://127.0.0.1:8000/api/v1/job/
{"jobs":{}}
ajvb@cloak:~/gocode/src/github.com/ajvb/kala$ curl http://127.0.0.1:8000/api/v1/job/ -d '{"epsilon": "PT5S", "command": "bash /home/ajvb/gocode/src/github.com/ajvb/kala/examples/example-kala-commands/example-command.sh", "name": "test_job", "schedule": "R2/2015-06-04T19:25:16.828696-07:00/PT10S"}'
{"id":"93b65499-b211-49ce-57e0-19e735cc5abd"}
ajvb@cloak:~/gocode/src/github.com/ajvb/kala$ curl http://127.0.0.1:8000/api/v1/job/
{"jobs":{"93b65499-b211-49ce-57e0-19e735cc5abd":{"name":"test_job","id":"93b65499-b211-49ce-57e0-19e735cc5abd","command":"bash /home/ajvb/gocode/src/github.com/ajvb/kala/examples/example-kala-commands/example-command.sh","owner":"","disabled":false,"dependent_jobs":null,"parent_jobs":null,"schedule":"R2/2015-06-04T19:25:16.828696-07:00/PT10S","retries":0,"epsilon":"PT5S","success_count":0,"last_success":"0001-01-01T00:00:00Z","error_count":0,"last_error":"0001-01-01T00:00:00Z","last_attempted_run":"0001-01-01T00:00:00Z","next_run_at":"2015-06-04T19:25:16.828794572-07:00"}}}
```

## /job/{id}

## /job/stats/{id}

## /job/start/{id}

## /stats


# Documentation

TODO

# Contributing

TODO

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
