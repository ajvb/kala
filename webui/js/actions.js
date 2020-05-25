var actions = {

  toggleJobDisabled: function(id) {
    var job = store.data.jobs[id];
    if (job) {
      job.disabled ? enableJob(job.id) : disableJob(job.id);
    }
  },

  disableJob: function(id) {
    kala.disableJob(id).then(function() {
      store.do('setJobDisabled', id, true);
    });
  },

  enableJob: function(id) {
    kala.enableJob(id).then(function() {
      store.do('setJobDisabled', id, false);
    });
  },

  runJob: function(id) {
    kala.runJob(id)
      .then(function() {
        return kala.getJob(id);
      }).then(function(job) {
      store.do('setJob', job);
    })
  },

  deleteJob: function(id) {
    kala.deleteJob(id)
      .then(function() {
        store.do('deleteJob', id);
      });
  },

  createJob: function(body) {
    kala.createJob(body)
      .then(function(id) {
        return kala.getJob(id);
      }).then(function(job) {
      store.do('setJob', job);
    });
  },

  getMetrics: function() {
    store.do('startLoading');
    return Promise.all([
      kala.metrics(),
      new Promise(function(resolve) {
        setTimeout(resolve, 1000)
      })
    ]).then(function(results) {
      var metrics = results[0];
      store.do('setMetrics', metrics);
    })
  },

  getJobs: function() {
    store.do('startLoading');
    return Promise.all([
      kala.getJobs(),
      new Promise(function(resolve) {
        setTimeout(resolve, 1000)
      })
    ]).then(function(results) {
      var resp = results[0];
      store.do('setJobs', resp.jobs);
    })
  },

  refresh: function(routeID) {
    if (routeID === 'jobs') {
      this.getJobs();
    } else if (routeID === 'metrics') {
      this.getMetrics();
    }
  },

}
