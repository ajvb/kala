#Kala

[![Circle CI](https://circleci.com/gh/ajvb/kala.svg?style=svg)](https://circleci.com/gh/ajvb/kala)

#### Currently in Alpha stage. Do not use in production enviorments.

Kala is a simplistic, modern, and performant job scheduler written in Go. It lives in a single binary and does not have any dependencies.

Kala was inspired by [Chronos](https://github.com/airbnb/chronos), developed by Airbnb, but the need for a Chronos for the rest of us. Chronos is built on top of Mesos, and
is fault tolerant and distributed by design. These are two features which Kala does not have, as it was built for smaller deployments.

It has a simple JSON over HTTP API, so it is language agnostic. It has Job Stats, Configurable Retries, uses ISO 8601 Date and Interval
notation, Dependant Jobs, and is Persistent (using BoltDB). Eventually it will support Redis as a Backend and have a Web UI.

#### Have any feedback or bugs to report?

Please create an issue within Github, or also feel free to email me at aj <at> ajvb.em

#### I need [fault tolerance, distributed-features, this to work at scale]

I recommend checking out [Chronos](https://github.com/airbnb/chronos). This is designed to be the Chronos for start-ups.

# Installing Kala

## Source

Step 0: Requires Go 1.0+ and git

Step 1: Clone this repo

```
git clone https://github.com/ajvb/kala.git
```

Step 2: Install Dependencies

```
cd kala && go get ./...
```

Step 3: Build the Kala binary

```
go build
```

Step 4: Move to somewhere in your $PATH

```
mv kala /usr/local/bin/
```

One liner:
```
git clone https://github.com/ajvb/kala.git && cd kala && go get ./... && go build && mv kala /usr/local/bin/
```


# Getting Started

Once you have installed Kala onto the machine you would like to use, you can follow the below steps to start using it.

To Run Kala:
```bash
ajvb$ kala run
2015/06/10 18:31:31 main.go:59:func·001 :: INFO 002 Starting server on port :8000...

ajvb$ kala run -p 2222
2015/06/10 18:31:31 main.go:59:func·001 :: INFO 002 Starting server on port :2222...
```

Kala runs on `127.0.0.1:8000` by default. You can easily test it out by curling the metrics path.

```bash
ajvb$ curl http://127.0.0.1:8000/api/v1/stats/
{"Stats":{"ActiveJobs":2,"DisabledJobs":0,"Jobs":2,"ErrorCount":0,"SuccessCount":0,"NextRunAt":"2015-06-04T19:25:16.82873873-07:00","LastAttemptedRun":"0001-01-01T00:00:00Z","CreatedAt":"2015-06-03T19:58:21.433668791-07:00"}}
```

Once its up in running, you can utilize curl or the official go client to interact with Kala. Also check out the examples directory.

### Examples of Usage

There are more examples in the examples directory within this repo. Currently its pretty messy. Feel free to submit a new example if you have one.

# Deployment

### Supervisord

After installing supervisord, open its config file (`/etc/supervisor/supervisord.conf` is the default usually) and add something like:

```
[program:kala]
command=kala run
autorestart=true
stdout_logfile=/var/log/kala.stdout.log
stderr_logfile=/var/log/kala.stderr.log
```

# API v1 Docs

All routes have a prefix of `/api/v1`

## Client Libraries

#### Official:
* Go - [client](https://github.com/ajvb/kala/tree/master/client) - Docs: http://godoc.org/github.com/ajvb/kala/client

## Job Data Struct

[Docs can be found here](http://godoc.org/github.com/ajvb/kala/job#Job)

## Job JSON Example

```
{
        "name":"test_job",
        "id":"93b65499-b211-49ce-57e0-19e735cc5abd",
        "command":"bash /home/ajvb/gocode/src/github.com/ajvb/kala/examples/example-kala-commands/example-command.sh",
        "owner":"",
        "disabled":false,
        "dependent_jobs":null,
        "parent_jobs":null,
        "schedule":"R2/2015-06-04T19:25:16.828696-07:00/PT10S",
        "retries":0,
        "epsilon":"PT5S",
        "success_count":0,
        "last_success":"0001-01-01T00:00:00Z",
        "error_count":0,
        "last_error":"0001-01-01T00:00:00Z",
        "last_attempted_run":"0001-01-01T00:00:00Z",
        "next_run_at":"2015-06-04T19:25:16.828794572-07:00"
}
```

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
```bash
ajvb$ curl http://127.0.0.1:8000/api/v1/job/
{"jobs":{}}
ajvb$ curl http://127.0.0.1:8000/api/v1/job/ -d '{"epsilon": "PT5S", "command": "bash /home/ajvb/gocode/src/github.com/ajvb/kala/examples/example-kala-commands/example-command.sh", "name": "test_job", "schedule": "R2/2015-06-04T19:25:16.828696-07:00/PT10S"}'
{"id":"93b65499-b211-49ce-57e0-19e735cc5abd"}
ajvb$ curl http://127.0.0.1:8000/api/v1/job/
{
    "jobs":{
        "93b65499-b211-49ce-57e0-19e735cc5abd":{
            "name":"test_job",
            "id":"93b65499-b211-49ce-57e0-19e735cc5abd",
            "command":"bash /home/ajvb/gocode/src/github.com/ajvb/kala/examples/example-kala-commands/example-command.sh",
            "owner":"",
            "disabled":false,
            "dependent_jobs":null,
            "parent_jobs":null,
            "schedule":"R2/2015-06-04T19:25:16.828696-07:00/PT10S",
            "retries":0,
            "epsilon":"PT5S",
            "success_count":0,
            "last_success":"0001-01-01T00:00:00Z",
            "error_count":0,
            "last_error":"0001-01-01T00:00:00Z",
            "last_attempted_run":"0001-01-01T00:00:00Z",
            "next_run_at":"2015-06-04T19:25:16.828794572-07:00"
        }
    }
}
```

## /job/{id}

This route accepts both a GET and a DELETE, and is based off of the id of the Job. Performing a GET request will return a full JSON object describing the Job.
Performing a DELETE will delete the Job.

Example:
```bash
ajvb$ curl http://127.0.0.1:8000/api/v1/job/93b65499-b211-49ce-57e0-19e735cc5abd
{"job":{"name":"test_job","id":"93b65499-b211-49ce-57e0-19e735cc5abd","command":"bash /home/ajvb/gocode/src/github.com/ajvb/kala/examples/example-kala-commands/example-command.sh","owner":"","disabled":false,"dependent_jobs":null,"parent_jobs":null,"schedule":"R2/2015-06-04T19:25:16.828696-07:00/PT10S","retries":0,"epsilon":"PT5S","success_count":0,"last_success":"0001-01-01T00:00:00Z","error_count":0,"last_error":"0001-01-01T00:00:00Z","last_attempted_run":"0001-01-01T00:00:00Z","next_run_at":"2015-06-04T19:25:16.828737931-07:00"}}
ajvb$ curl http://127.0.0.1:8000/api/v1/job/93b65499-b211-49ce-57e0-19e735cc5abd -X DELETE
ajvb$ curl http://127.0.0.1:8000/api/v1/job/93b65499-b211-49ce-57e0-19e735cc5abd
```

## /job/stats/{id}

Example:
```bash
ajvb$ curl http://127.0.0.1:8000/api/v1/job/stats/5d5be920-c716-4c99-60e1-055cad95b40f/
{"job_stats":[{"JobId":"5d5be920-c716-4c99-60e1-055cad95b40f","RanAt":"2015-06-03T20:01:53.232919459-07:00","NumberOfRetries":0,"Success":true,"ExecutionDuration":4529133}]}
```

## /job/start/{id}

Example:
```bash
ajvb$ curl http://127.0.0.1:8000/api/v1/job/start/5d5be920-c716-4c99-60e1-055cad95b40f/ -X POST
```

## /stats

Example:
```bash
ajvb$ curl http://127.0.0.1:8000/api/v1/stats/
{"Stats":{"ActiveJobs":2,"DisabledJobs":0,"Jobs":2,"ErrorCount":0,"SuccessCount":0,"NextRunAt":"2015-06-04T19:25:16.82873873-07:00","LastAttemptedRun":"0001-01-01T00:00:00Z","CreatedAt":"2015-06-03T19:58:21.433668791-07:00"}}
```

# Documentation

[Can be found here](http://godoc.org/github.com/ajvb/kala)

# Contributing

TODO

# TODO's

### Features
- [ ] Web UI
- [ ] Config file and/or CL flags
    - Port & Host (needs improvement)
    - Verbose/Debug logging
    - Default owner
- [ ] Error Reporting on job failure
- [ ] Remove dependance on external http library in client.

### For User
- [ ] Users Documentation
- [ ] Python Client Library
- [ ] Node Client Library
- [ ] Create single release binary

### For Contributors
- [ ] Contributors Documentation
- [ ] Continuous Integration

### Testing
- [ ] Add race detector to circleci
- [ ] Add API Test Suite
- [ ] Add edge case tests for job scheduling and job dependency

# Original Contributors and Contact

Original Author and Core Maintainer: AJ Bahnken / [@ajvbahnken](http://twitter.com/ajvbahnken) / aj@ajvb.me
Original Reviewers:
Sam Dolan / [@samdolan](https://github.com/samdolan/)
Steve Phillips / [@elimisteve](http://twitter.com/elimisteve)
