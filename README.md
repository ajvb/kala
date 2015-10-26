#Kala

[![Circle CI](https://circleci.com/gh/ajvb/kala.svg?style=svg)](https://circleci.com/gh/ajvb/kala)
[![Coverage Status](https://coveralls.io/repos/ajvb/kala/badge.svg?branch=master&service=github)](https://coveralls.io/github/ajvb/kala?branch=master)

#### Donate

[![Flattr this git repo](http://api.flattr.com/button/flattr-badge-large.png)](https://flattr.com/submit/auto?user_id=ajvb&url=https://github.com/ajvb/kala&title=Kala&language=&tags=github&category=software)

#### Currently in Alpha stage. Do not use in production environments.

Kala is a simplistic, modern, and performant job scheduler written in Go. It lives in a single binary and does not have any dependencies.

Kala was inspired by the desire for a simpler [Chronos](https://github.com/airbnb/chronos) (developed by Airbnb). Kala is Chronos for the rest of us.

It has a simple JSON over HTTP API, so it is language agnostic. It has Job Stats, Configurable Retries, uses ISO 8601 Date and Interval
notation, Dependant Jobs, and is Persistent (using BoltDB). Eventually it will have a Web UI.

#### Have any feedback or bugs to report?

Please create an issue within Github, or also feel free to email me at aj <at> ajvb.me


Mailing List: https://groups.google.com/forum/#!forum/kala-scheduler


#### I need [fault tolerance, distributed-features, this to work at scale]

I recommend checking out [Chronos](https://github.com/airbnb/chronos). This is designed to be the Chronos for start-ups.

## Installing Kala

*Requires Go 1.0+ and git*

1. Get Kala

	```
	go get github.com/ajvb/kala
	```

2. Run Kala

	```
	kala run
	```

## Development

1. Change directory to Kala source

	```
	cd $GOPATH/src/github.com/ajvb/kala
	```

2. Install godep command

	```
	go get github.com/tools/godep
	```

3. Restore Godeps

	```
	godep restore
	```

4. Build the local Kala binary

	```
	go build
	```

5. Run local Kala

	```
	./kala run
	```

6. **Optional:** Replace the preinstalled Kala with local Kala

	```
	go install
	```


# Getting Started

Once you have installed Kala onto the machine you would like to use, you can follow the below steps to start using it.

To Run Kala:
```bash
$ kala run
2015/06/10 18:31:31 main.go:59:func·001 :: INFO 002 Starting server on port :8000...

$ kala run -p 2222
2015/06/10 18:31:31 main.go:59:func·001 :: INFO 002 Starting server on port :2222...
```

Kala uses BoltDB by default for the job database, however you can also use Redis by using the jobDB and jobDBAddress params:

```bash
kala run --jobDB=redis --jobDBAddress=127.0.0.1:6379
```

Kala runs on `127.0.0.1:8000` by default. You can easily test it out by curling the metrics path.

```bash
$ curl http://127.0.0.1:8000/api/v1/stats/
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

### Docker 

If you have docker installed, you can build the dockerfile in this directory with
```docker build -t kala .```
and run it as a daemon with:
```docker run -it -d -p 8000:8000 kala```

# API v1 Docs

All routes have a prefix of `/api/v1`

## Client Libraries

#### Official:
* Go - [client](https://github.com/ajvb/kala/tree/master/client) - Docs: http://godoc.org/github.com/ajvb/kala/client

Install using:
`go get github.com/ajvb/kala/client`

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
$ curl http://127.0.0.1:8000/api/v1/job/
{"jobs":{}}
$ curl http://127.0.0.1:8000/api/v1/job/ -d '{"epsilon": "PT5S", "command": "bash /home/ajvb/gocode/src/github.com/ajvb/kala/examples/example-kala-commands/example-command.sh", "name": "test_job", "schedule": "R2/2015-06-04T19:25:16.828696-07:00/PT10S"}'
{"id":"93b65499-b211-49ce-57e0-19e735cc5abd"}
$ curl http://127.0.0.1:8000/api/v1/job/
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
$ curl http://127.0.0.1:8000/api/v1/job/93b65499-b211-49ce-57e0-19e735cc5abd
{"job":{"name":"test_job","id":"93b65499-b211-49ce-57e0-19e735cc5abd","command":"bash /home/ajvb/gocode/src/github.com/ajvb/kala/examples/example-kala-commands/example-command.sh","owner":"","disabled":false,"dependent_jobs":null,"parent_jobs":null,"schedule":"R2/2015-06-04T19:25:16.828696-07:00/PT10S","retries":0,"epsilon":"PT5S","success_count":0,"last_success":"0001-01-01T00:00:00Z","error_count":0,"last_error":"0001-01-01T00:00:00Z","last_attempted_run":"0001-01-01T00:00:00Z","next_run_at":"2015-06-04T19:25:16.828737931-07:00"}}
$ curl http://127.0.0.1:8000/api/v1/job/93b65499-b211-49ce-57e0-19e735cc5abd -X DELETE
$ curl http://127.0.0.1:8000/api/v1/job/93b65499-b211-49ce-57e0-19e735cc5abd
```

## /job/stats/{id}

Example:
```bash
$ curl http://127.0.0.1:8000/api/v1/job/stats/5d5be920-c716-4c99-60e1-055cad95b40f/
{"job_stats":[{"JobId":"5d5be920-c716-4c99-60e1-055cad95b40f","RanAt":"2015-06-03T20:01:53.232919459-07:00","NumberOfRetries":0,"Success":true,"ExecutionDuration":4529133}]}
```

## /job/start/{id}

Example:
```bash
$ curl http://127.0.0.1:8000/api/v1/job/start/5d5be920-c716-4c99-60e1-055cad95b40f/ -X POST
```

## /stats

Example:
```bash
$ curl http://127.0.0.1:8000/api/v1/stats/
{"Stats":{"ActiveJobs":2,"DisabledJobs":0,"Jobs":2,"ErrorCount":0,"SuccessCount":0,"NextRunAt":"2015-06-04T19:25:16.82873873-07:00","LastAttemptedRun":"0001-01-01T00:00:00Z","CreatedAt":"2015-06-03T19:58:21.433668791-07:00"}}
```

# Documentation

[Contributor Documentation can be found here](http://godoc.org/github.com/ajvb/kala)

## Dependent Jobs

### How to add a dependent job

Check out this [example for how to add dependent jobs](https://github.com/ajvb/kala/blob/master/examples/example_dependent_jobs.py) within a python script.

### Notes on Dependent Jobs

* Dependent jobs follow a rule of First In First Out
* A child will always have to wait until a parent job finishes before it runs
* A child will not run if its parent job does not.
* If a child job is disabled, it's parent job will still run, but it will not.
* If a child job is deleted, it's parent job will continue to stay around.
* If a parent job is deleted, unless its child jobs have another parent, they will be deleted as well.

# Contributing

TODO

# TODO's

### For User
- [ ] Python Client Library
- [ ] Node Client Library
- [ ] Create single release binary

### For Contributors
- [ ] Contributors Documentation

# Original Contributors and Contact

Original Author and Core Maintainer:
    * AJ Bahnken / [@ajvbahnken](http://twitter.com/ajvbahnken) / aj@ajvb.me

Original Reviewers:
    * Sam Dolan / [@samdolan](https://github.com/samdolan/)
    * Steve Phillips / [@elimisteve](http://twitter.com/elimisteve)
