import requests
import simplejson
import os
from time import sleep
from datetime import datetime, timedelta
from dateutil.tz import tzlocal

API_URL = "http://127.0.0.1:8000/api/v1/job/"

data = {
    "name": "test_job",
    "command": "bash " + os.path.dirname(os.path.realpath(__file__)) + "../example-kala-commands/example-command.sh",
    "epsilon": "PT5S",
}

dt = datetime.isoformat(datetime.now(tzlocal()) + timedelta(0, 10))
data["schedule"] = "%s/%s/%s" % ("R2", dt, "PT10S")

if __name__ == "__main__":
    print "Sending request to %s" % API_URL
    print "Payload is: %s" % data

    r = requests.post(API_URL, data=simplejson.dumps(data))

    print "\n\n CREATING \n"
    job_id = simplejson.loads(r.content)['id']
    print "Job was created with an id of %s" % job_id

    print "\n\n GETTING \n"
    m = requests.get(API_URL + job_id)
    print m.content

    print "\n\n DELETING \n"

    print "Waiting to delete...."
    sleep(21)
    n = requests.delete(API_URL + job_id)

    print n.__dict__
