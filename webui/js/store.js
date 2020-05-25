var store = new Reef.Store({
  data: {
    jobs: {},
    jobDetail: null,
    metrics: {},
    loading: false,
  },
  setters: {
    startLoading: function(props) {
      props.loading = true;
    },
    setJobs: function(props, jobs) {
      props.jobs = jobs;
      props.loading = false;
    },
    setJob: function(props, job) {
      props.jobs[job.id] = job;
    },
    showJobDetail: function(props, jobID) {
      props.jobDetail = props.jobs[jobID];
    },
    clearJobDetail: function(props) {
      props.jobDetail = null;
    },
    setJobDisabled: function(props, jobID, disabled) {
      if (props.jobs[jobID]) {
        props.jobs[jobID].disabled = disabled;
        if (props.jobDetail.id === jobID) {
          props.jobDetail.disabled = disabled;
        }
      }
    },
    deleteJob: function(props, jobID) {
      if (props.jobs[jobID]) {
        delete props.jobs[jobID];
        if (props.jobDetail.id === jobID) {
          props.jobDetail = null;
        }
      }
    },
    setMetrics: function(props, metrics) {
      props.metrics = metrics;
      props.loading = false;
    },
  },
});
