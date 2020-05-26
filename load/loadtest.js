import http from "k6/http";
import { check, sleep } from "k6";
import { Trend } from 'k6/metrics';

export let options = {
    stages: [
        // Ramp-up from 1 to 5 VUs in 10s
        {
            duration: "10s",
            target: 50
        },

        // Stay at rest on 5 VUs for 5s
        {
            duration: "5s",
            target: 50
        },

        // Ramp-down from 5 to 0 VUs for 5s
        {
            duration: "5s",
            target: 0
        },
        // Ramp-down from 5 to 0 VUs for 5s
        {
            duration: "10s",
            target: 1
        }
    ]
};

let jobs = new Trend('jobs');
let activeJobs = new Trend('active_jobs');
let errorCount = new Trend('error_count');
let successCount = new Trend('success_count');

var count = 0;

function getSchedule(now) {
    now.setSeconds(now.getSeconds() + 1);
    return 'R5/' + now.toISOString() + '/PT5S';
}

export default function() {
    let payload = JSON.stringify({
        name: 'job' + (count++),
        schedule: getSchedule(new Date()),
        command: "bash -c 'date'"
    });
    let write = http.post("http://localhost:8000/api/v1/job/", payload);
    check(write, {
        "status is ok": (r) => r.status === 201
    });
    sleep(1);
    let read = http.get("http://localhost:8000/api/v1/job/");
    check(read, {
        "status is ok": (r) => r.status === 200
    });
    sleep(1);
    let stats = http.get("http://localhost:8000/api/v1/stats/");
    check(stats, {
        "status is ok": (r) => r.status === 200
    });
    let parsed = JSON.parse(stats.body);
    jobs.add(parsed.Stats.jobs);
    activeJobs.add(parsed.Stats.active_jobs);
    errorCount.add(parsed.Stats.error_count);
    successCount.add(parsed.Stats.success_count);
    sleep(1);
}
