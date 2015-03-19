#!/bin/bash
http http://127.0.0.1:8000/api/v1/job/ name=test_job command="touch lol" schedule=R2/2015-03-18T21:48:30-07:00/PT5S
