import requests
import simplejson
import os
from datetime import datetime, timedelta
from dateutil.tz import tzlocal



API_URL = "http://127.0.0.1:8000/api/v1/job/"

data = {
    "name": "test_job",
    "command": "bash " + os.path.dirname(os.path.realpath(__file__)) + "/example-kala-commands/example-command.sh"
}

# Add schedule time of repeat twice, start 30 seconds from now, and run every 10 seconds.
# Will take 50 seconds for all three commands to run.
dt = datetime.isoformat(datetime.now(tzlocal()) + timedelta(0, 30))

data["schedule"] = "%s/%s/%s" % ("R2", dt, "PT10S")

if __name__ == "__main__":
    print "Sending request to %s" % API_URL
    print "Payload is: %s" % data

    r = requests.post(API_URL, data=simplejson.dumps(data))

    print r.__dict__
