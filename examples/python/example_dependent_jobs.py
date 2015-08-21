import requests
import simplejson
import os
from time import sleep
from datetime import datetime, timedelta
from dateutil.tz import tzlocal

API_URL = "http://127.0.0.1:8000/api/v1/job/"

scheduled_start = datetime.isoformat(datetime.now(tzlocal()) + timedelta(0, 10))
parent_job = {
    "name": "test_job",
    "command": "bash " + os.path.dirname(os.path.realpath(__file__)) + "/../example-kala-commands/example-command.sh",
    "epsilon": "PT5S",
    "schedule": "%s/%s/%s" % ("R2", scheduled_start, "PT10S"),
}
child_job = {
    "name": "my_child_job",
    "command": "bash " + os.path.dirname(os.path.realpath(__file__)) + "/../example-kala-commands/example-command.sh",
    "epsilon": "PT5S",
}

if __name__ == "__main__":
    print "Sending request to %s" % API_URL

    r = requests.post(API_URL, data=simplejson.dumps(parent_job))

    job_id = simplejson.loads(r.content)['id']
    print "Parent Job was created with an id of %s" % job_id

    # Make sure to add the parent_job's id to the payload.
    child_job["parent_jobs"] = [job_id]

    print "Creating child job..."
    new_req = requests.post(API_URL, data=simplejson.dumps(child_job))

    child_job_id = simplejson.loads(new_req.content)['id']
    print "Child Job was created with an id of %s" % child_job_id

    print "Waiting to delete...."
    sleep(21)
    n = requests.delete(API_URL + child_job_id)
    n = requests.delete(API_URL + job_id)

    print n.__dict__
