var actions = (function() {

  var reqMsg = 'This field is required.';
  function hasRequiredFields(fields) {
    if (!fields.name) {
      store.do('setCreateErr', 'name', reqMsg);
      return false;
    }
    if (fields.type === 0 && !fields.command) {
      store.do('setCreateErr', 'command', reqMsg);
      return false;
    }
    return true;
  }

  var actions = {

    toggleJobDisabled: function(id) {
      var job = store.data.jobs[id];
      if (job) {
        job.disabled ? actions.enableJob(job.id) : actions.disableJob(job.id);
      }
    },

    disableJob: function(id) {
      store.do('startLoading');
      kala.disableJob(id).then(function() {
        store.do('setJobDisabled', id, true);
      });
    },

    enableJob: function(id) {
      store.do('startLoading');
      kala.enableJob(id).then(function() {
        store.do('setJobDisabled', id, false);
      });
    },

    runJob: function(id) {
      store.do('startLoading');
      Promise.all([
        kala.runJob(id),
        new Promise(function(resolve) {
          setTimeout(resolve, 750)
        })
      ]).then(function(results) {
        return kala.getJob(id);
      }).then(function(job) {
        store.do('setJob', job);
      });
    },

    deleteJob: function(id) {
      store.do('startLoading');
      Promise.all([
        kala.deleteJob(id),
        new Promise(function(resolve) {
          setTimeout(resolve, 750)
        })
      ])
        .then(function() {
          store.do('deleteJob', id);
        });
    },

    createJob: function(job) {
      store.do('startLoading');
      return kala.createJob(job)
        .then(function(id) {
          return (new Promise(function(resolve) {
            setTimeout(resolve, 750)
          }))
            .then(function() {
              return kala.getJob(id);
            })
        })
        .then(function(job) {
          store.do('setJob', job);
        })
        .catch(function(ex) {
          console.error(ex);
        });
    },

    getMetrics: function() {
      store.do('startLoading');
      return Promise.all([
        kala.metrics(),
        new Promise(function(resolve) {
          setTimeout(resolve, 750)
        })
      ])
        .then(function(results) {
          var metrics = results[0];
          store.do('setMetrics', metrics);
        })
    },

    getJobs: function() {
      store.do('startLoading');
      return Promise.all([
        kala.getJobs(),
        new Promise(function(resolve) {
          setTimeout(resolve, 750)
        })
      ])
        .then(function(results) {
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

    submitCreateJob: function(selector) {
      store.do('setCreateErr', null);
      var job = {};
      var resetForm;
      try {
        var form = new FormData(document.querySelector(selector));
        var entries = form.entries();
        var iter;
        while (!(iter = entries.next()).done) {
          var key = iter.value[0];
          var val = iter.value[1];
          if (key === 'resume_at_next_scheduled_time') {
            job[key] = (val === 'on') ? true : false;
          } else if (key === 'jobname') {
            job['name'] = val || '';
          } else if (key === 'doReset') {
            resetForm = !!val;
          } else if (key === 'type' || key === 'timeout' || key === 'retries') {
            job[key] = val && parseInt(val) || 0;
          } else if (key === 'headers' && val) {
            try {
              job[key] = JSON.parse(val);
            } catch ( err ) {
              store.do('setCreateErr', 'headers', err);
            }
          } else if (key === 'expected_response_codes') {
            job[key] = val.split(',').filter(function(chunk) {
              return !!chunk
            }).map(function(code) {
              return parseInt(code);
            });
          } else if (val) {
            job[key] = val;
            if (/(timeout|body|method|url|headers)/i.test(key)) {
              job['remote_properties'] = job['remote_properties'] || {};
              job['remote_properties'][key] = job[key];
            }
          }
        }
      } catch ( err ) {
        console.error(err);
        return false;
      }
      console.debug(job);
      if (!hasRequiredFields(job)) {
        return false;
      }
      actions.createJob(job)
        .then(function() {
          if (resetForm) {
            document.querySelector(selector).reset();
          }
        });
      return false;
    },

  }

  return actions;

})();
