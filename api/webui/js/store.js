var store = new Reef.Store({
  data: {

    jobs: {},
    jobDetail: null,
    metrics: {},
    loading: false,

    createType: 'local',
    createErr: {},
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
      if (props.jobDetail && props.jobDetail.id === job.id) {
        props.jobDetail = job;
      }
      props.loading = false;
    },
    showJobDetail: function(props, jobID) {
      props.jobDetail = props.jobs[jobID];
    },
    clearJobDetail: function(props) {
      props.jobDetail = null;
    },
    setJobDisabled: function(props, jobID, disabled) {
      props.loading = false;
      if (props.jobs[jobID]) {
        props.jobs[jobID].disabled = disabled;
        if (props.jobDetail.id === jobID) {
          props.jobDetail.disabled = disabled;
        }
      }
    },
    deleteJob: function(props, jobID) {
      props.loading = false;
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
    setCreateTypeLocal: function(props) {
      props.createType = 'local'
    },
    setCreateTypeRemote: function(props) {
      props.createType = 'remote'
    },
    setCreateErr: function(props, name, err) {
      props.createErr = {};
      if (name) {
        props.createErr[name] = err;
      }
    }
  },
});
