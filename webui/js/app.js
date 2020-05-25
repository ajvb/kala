Reef.debug(true);

var app = new Reef('#app', {
  store: store,
  router: router,
  template: function(props, route) {
    var activeForRoute = function(routeID) {
      return route && route.id === routeID ? '' : 'is-hidden'
    }
    return html`
      <div id="nav"></div>
      <section class="section">
        <div class="container">
          <h1 class="title">
            ${route.title}
            <span class="is-pulled-right">
              <button class="button is-rounded is-info is-outlined" onclick="actions.refresh('${route.id}')">
                <span class="icon">
                  <i class="fas fa-sync"></i>
                </span>
                <span>Refresh</span>
              </button>
            </span>
          </h1>
          <div class="box">
            <div class="container" style="min-height: 300px">
              <div id="jobsTable" class="${activeForRoute('jobs')}"></div>
              <div id="metricsPanel" class="${activeForRoute('metrics')}"></div>
              <div class="loader-wrapper ${props.loading ? 'is-active' : ''}">
                <div class="loader is-loading"></div>
              </div>
            </div>
          </div>
        </div>
      </section>
      <div id="jobDetailModal"></div>
    `
  },
});

var navbar = new Reef('#nav', {
  store: store,
  router: router,
  attachTo: app,
  template: function(props, route) {
    var activeForRoute = function(routeID) {
      return route && route.id === routeID ? 'is-active' : ''
    }
    return html`
      <nav class="navbar is-dark" role="navigation" aria-label="main navigation">
        <div class="navbar-brand">
          <a class="navbar-item" href="/webui/">
            <img src="logo.png" height="28">
          </a>

          <a role="button" class="navbar-burger burger" aria-label="menu" aria-expanded="false" data-target="navbarBasicExample">
            <span aria-hidden="true"></span>
            <span aria-hidden="true"></span>
            <span aria-hidden="true"></span>
          </a>
        </div>

        <div class="navbar-menu">
          <div class="navbar-start">
            <a class="navbar-item ${activeForRoute('metrics')}" href="metrics">
              Metrics
            </a>
            <a class="navbar-item ${activeForRoute('jobs')}" href="jobs">
              Jobs
            </a>
            <a class="navbar-item ${activeForRoute('create')}" href="jobs">
              Create
            </a>
          </div>

          <div class="navbar-end">
            <div class="navbar-item">
              <div class="buttons">
                <a class="button is-link" href="https://github.com/ajvb/kala">
                  <span class="icon">
                    <i class="fab fa-github"></i>
                  </span>
                  <span>GitHub</span>
                </a>
              </div>
            </div>
          </div>
        </div>
      </nav>
    `
  },
});

var jobsTable = new Reef('#jobsTable', {
  store: store,
  attachTo: app,
  template: function(props, route) {
    var rows = Object.entries(props.jobs || {}).reduce(function(acc, entry) {
      var id = entry[0];
      var job = entry[1];
      return acc + html`
        <tr>
          <td><a onclick="return store.do('showJobDetail', '${id}')">${id}</a></td>
          <td>${job.name}</td>
          <td>${job.owner}</td>
          <td>${jobType(job.type)}</td>
          <td>${job.disabled ? 'Disabled' : 'Enabled'}</td>
          <td>${job.stats ? job.stats.length : 0}</td>
          <td>${job.is_done}</td>
        </tr>
      `
    }, '');
    return html`
      <table class="table is-fullwidth is-striped">
        <thead>
          <tr>
            <th>ID</th>
            <th>Name</th>
            <th>Owner</th>
            <th>Type</th>
            <th>Status</th>
            <th>Runs</th>
            <th>Done</th>
          </tr>
        </thead>
        <tbody>
          ${rows}
        </tbody>
      </table>
    `
  },
});

var jobDetail = new Reef('#jobDetailModal', {
  store: store,
  attachTo: app,
  template: function(props, route) {
    if (props.jobDetail) {
      var id = props.jobDetail && props.jobDetail.id;
      return html`
        <div class="modal ${props.jobDetail ? 'is-active' : ''}">
          <div class="modal-background" onclick="store.do('clearJobDetail')"></div>
           <div class="modal-card">
            <header class="modal-card-head">
              <p class="modal-card-title">
                ${props.jobDetail.name}
                <span class="tag ${props.jobDetail.disabled ? 'is-warning' : 'is-success'}">
                  ${props.jobDetail.disabled ? 'Disabled' : 'Enabled'}
                </span>
              </p>
              <button class="delete" aria-label="close" onclick="store.do('clearJobDetail')"></button>
            </header>
            <section class="modal-card-body">
              <pre>${JSON.stringify(props.jobDetail, undefined, 4)}</pre>
            </section>
            <footer class="modal-card-foot">
              <button class="button is-primary">Stats</button>
              <button class="button is-primary" onclick="actions.toggleJobDisabled('${id}')">
                ${props.jobDetail.disabled ? 'Enable' : 'Disable'}
              </button>
              <button class="button is-primary" onclick="actions.runJob('${id}')">Run Manually</button>
              <button class="button is-danger" onclick="actions.deleteJob('${id}')">Delete</button>
            </footer>
          </div>
          <button class="modal-close is-large" aria-label="close" onclick="store.do('clearJobDetail')"></button>
        </div>
      `
    } else {
      return '';
    }
  },
});

var metricsPanel = new Reef('#metricsPanel', {
  store: store,
  attachTo: app,
  template: function(props, route) {
    return html`
      <table class="table is-bordered">
        <thead>
          <tr>
            <th>Metric</th>
            <th>Value</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>Active Jobs</td>
            <td>${props.metrics.active_jobs || '0'}</td>
          </tr>
          <tr>
            <td>Disabled Jobs</td>
            <td>${props.metrics.disabled_jobs || '0'}</td>
          </tr>
          <tr>
            <td>Jobs</td>
            <td>${props.metrics.jobs || '0'}</td>
          </tr>
          <tr>
            <td>Error Count</td>
            <td>${props.metrics.error_count || '0'}</td>
          </tr>
          <tr>
            <td>Success Count</td>
            <td>${props.metrics.success_count || '0'}</td>
          </tr>
          <tr>
            <td>Next Run At</td>
            <td>${props.metrics.next_run_at}</td>
          </tr>
          <tr>
            <td>Last Attempted Run At</td>
            <td>${props.metrics.last_attempted_run}</td>
          </tr>
          <tr>
            <td>Metrics Generated At</td>
            <td>${props.metrics.created}</td>
          </tr>
        </tbody>
      </table>
    `
  },
});

// Initial data load
actions.getJobs()
  .then(function() {
    actions.getMetrics()
  });
