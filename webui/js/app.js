Reef.debug(true);

var app = new Reef('#app', {
    store: store,
    router: router,
    template: function(props, route) {
        var activeForRoutes = function() {
            var args = Array.prototype.slice.call(arguments);
            return args.some(function(routeID) {
                return routeID === route.id
            }) ? '' : 'is-hidden'
        }
        return html`
      <div id="nav"></div>
      <section class="section">
        <div class="container">
          <h1 class="title">
            ${route.title} ${route.id === 'jobs' ? ' - ' + Object.keys(props.jobs).length : ''}
            <span class="is-pulled-right ${activeForRoutes('jobs', 'metrics')}">
              <button class="button is-rounded is-info is-outlined" onclick="actions.refresh('${route.id}')" ${props.loading && 'disabled'}>
                <span class="icon">
                  <i class="fas fa-sync"></i>
                </span>
                <span>Refresh</span>
              </button>
            </span>
          </h1>
          <div class="box">
            <div class="container" style="min-height: 300px">
              <div id="jobsTable" class="${activeForRoutes('jobs')}"></div>
              <div id="metricsPanel" class="${activeForRoutes('metrics')}"></div>
              <div id="createPage" class="${activeForRoutes('create')}"></div>
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
            return route.id === routeID ? 'is-active' : ''
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
            <a class="navbar-item ${activeForRoute('create')}" href="create">
              Create
            </a>
          </div>

          <div class="navbar-end">
            <div class="navbar-item">
              <div class="buttons">
                <a class="button is-white is-outlined" href="https://bitbucket.org/nextiva/nextkala">
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
        var rows = Object.entries(props.jobs || {}).reduce(function(acc, entry, idx) {
            var id = entry[0];
            var job = entry[1];
            return acc + html`
        <tr>
          <td>${idx + 1}</td>
          <td><a onclick="return store.do('showJobDetail', '${id}')">${id}</a></td>
          <td>${job.name}</td>
          <td>${job.owner}</td>
          <td>${jobType(job.type)}</td>
          <td>${job.disabled ? 'Disabled' : 'Enabled'}</td>
          <td>${job.metadata ? job.metadata.success_count + '/' + job.metadata.number_of_finished_runs : 0}</td>
          <td>${job.is_done}</td>
        </tr>
      `
        }, '');
        return html`
      <table class="table is-fullwidth is-striped">
        <thead>
          <tr>
            <th></th>
            <th>ID</th>
            <th>Name</th>
            <th>Owner</th>
            <th>Type</th>
            <th>Status</th>
            <th>Success</th>
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
              <!-- <button class="button is-primary">Stats</button> -->
              <button class="button is-primary" onclick="actions.toggleJobDisabled('${id}')">
                ${props.jobDetail.disabled ? 'Enable' : 'Disable'}
              </button>
              <button class="button is-primary" onclick="actions.runJob('${id}')" ${props.jobDetail.disabled && 'disabled'}>Run Manually</button>
              <button class="button is-danger" onclick="actions.deleteJob('${id}')">Delete</button>
            </footer>
            <div class="loader-wrapper ${props.loading ? 'is-active' : ''}">
              <div class="loader is-loading"></div>
            </div>
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

var createPanel = new Reef('#createPage', {
    store: store,
    attachTo: app,
    template: function(props, route) {
        var detailPanelForType = function(createType) {
            var err = function(fieldName, output) {
                var err = props.createErr[fieldName]
                return err ? (output || err) : ''
            }
            if (createType === 'local') {
                return html`
          <div class="field">
            <label class="label">Command</label>
            <div class="control">
              <input class="input ${props.createErr['command'] && 'is-danger'}" type="text" placeholder="bash -c 'date'" name="command">
            </div>
            <p class="help is-danger">${props.createErr['command'] || ''}</p>
          </div>
        `
            } else if (createType === 'remote') {
                var methods = ["GET", "POST", "HEAD", "PUT", "DELETE", "CONNECT", "OPTIONS", "PATCH", "TRACE"].reduce(function(acc, method, idx) {
                    return acc + html`<option value="${method}" ${idx === 0 && 'defaultSelected'}>${method}</option>`
                }, '')
                return html`
          <fieldset>
            <div class="field">
              <label class="label">URL</label>
              <div class="control">
                <input class="input" type="url" name="remote_properties.url">
              </div>
            </div>
            <div class="field">
              <label class="label">Method</label>
              <div class="control">
                <div class="select">
                  <select name="remote_properties.method">
                    ${methods}
                  </select>
                </div>
              </div>
            </div>
            <div class="field">
              <label class="label">Headers</label>
              <div class="control">
                <textarea rows="3" class="textarea ${props.createErr['headers'] && 'is-danger'}" type="text" name="remote_properties.headers" placeholder="{&quot;key&quot;: [&quot;val1&quot;, &quot;val2&quot;]}"></textarea>
              </div>
              <p class="help is-danger">${props.createErr['headers'] || ''}</p>
            </div>
            <div class="field">
              <label class="label">Body</label>
              <div class="control">
                <textarea rows="6" class="textarea" type="text" name="remote_properties.body"></textarea>
              </div>
            </div>
            <div class="field">
              <label class="label">Timeout</label>
              <div class="control">
                <input class="input" type="number" placeholder="seconds" name="remote_properties.timeout">
              </div>
            </div>
            <div class="field">
              <label class="label">Expected Response Codes</label>
              <div class="control">
                <input class="input" type="text" placeholder="200,201,204" name="remote_properties.expected_response_codes">
              </div>
            </div>
          </fieldset>
        `
            } else {
                return html`<div>Unsupported type!</div>`
            }
        }
        var checkedForType = function(t) {
            return props.createType === t ? 'checked="checked"' : ''
        }
        return html`
      <form id="createForm" onsubmit="return actions.submitCreateJob('#createForm')">
        <div class="columns">
          <div class="column is-one-third">
            <div class="field">
              <label class="label">Job Name</label>
              <div class="control">
                <input class="input ${props.createErr['name'] && 'is-danger'}" type="text" name="jobname">
              </div>
              <p class="help is-danger">${props.createErr['name'] || ''}</p>
            </div>
            <div class="field">
              <div class="control">
                <label class="radio">
                  <input type="radio" name="type" value="0" onchange="store.do('setCreateTypeLocal')" ${checkedForType('local')}>
                  Local
                </label>
                <label class="radio">
                  <input type="radio" name="type" value="1" onchange="store.do('setCreateTypeRemote')" ${checkedForType('remote')}>
                  Remote
                </label>
              </div>
            </div>
            <div class="field">
              <label class="label">Owner's Email</label>
              <div class="control">
                <input class="input" type="text" name="owner">
              </div>
            </div>
            <div class="field">
              <label class="label">Schedule</label>
              <div class="control">
                <input class="input" type="text" name="schedule">
              </div>
            </div>
            <div class="field">
              <label class="label">Epsilon</label>
              <div class="control">
                <input class="input" type="text" name="epsilon">
              </div>
            </div>
            <div class="field">
              <label class="label">Retries</label>
              <div class="control">
                <input class="input" type="number" name="retries">
              </div>
            </div>
            <div class="field">
              <div class="control">
                <label class="checkbox">
                  <input type="checkbox" name="resume_at_next_scheduled_time">
                  Resume At Next Scheduled Time
                </label>
              </div>
            </div>
            <div class="field">
              <div class="control">
                <label class="checkbox">
                  <input type="checkbox" name="doReset" defaultChecked>
                  Reset Form
                </label>
              </div>
            </div>
            <button class="button is-success" type="submit">Create</button>
          </div>
          <div class="column">
            ${detailPanelForType(props.createType)}
          </div>
        </div>
      </form>
    `
    },
});

// Initial data load
actions.getJobs()
    .then(function() {
        actions.getMetrics()
    });
