Reef.debug(true);

var store = new Reef.Store({
  data: {
    jobs: {},
    jobDetail: null,
  },
  setters: {
    setJobs: function(props, jobs) {
      props.jobs = jobs;
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
  },
});

var app = new Reef('#app', {
  store: store,
  router: router,
  template: function() {/*
    <div id="nav"></div>
    <section class="section">
      <div class="container">
        <h1 class="title">
          ${routeTitle}
        </h1>
        <div id="jobsTable"></div>
      </div>
    </section>
    <div id="jobDetailModal"></div>
  */}.extractComment().propsInjector(function(props, route){
    return Object.assign(props, {
      routeTitle: route.title,
    })
  }),
});

var navbar = new Reef('#nav', {
  store: store,
  attachTo: app,
  template: function(){/*
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
          <a class="navbar-item" href="metrics">
            Metrics
          </a>
          <a class="navbar-item" href="jobs">
            Jobs
          </a>
          <div class="navbar-item">
            <a class="button is-primary">
              <strong>Create</strong>
            </a>
          </div>
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
  */}.extractComment().propsInjector()
});

var jobsTable = new Reef('#jobsTable', {
  store: store,
  attachTo: app,
  template: function() {/*
    <table class="table">
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
  */}.extractComment().propsInjector(function(props){
    var tmpl = function(){/*
      <tr>
        <td><a onclick="return store.do('showJobDetail', '${id}')">${id}</a></td>
        <td>${name}</td>
        <td>${owner}</td>
        <td>${type}</td>
        <td>${disabled}</td>
        <td>${runs}</td>
        <td>${done}</td>
      </tr>
    */}.extractComment().propsInjector()
    props.rows = Object.entries(props.jobs).reduce(function(acc, entry){
      var record = {
        id: entry[0],
        name: entry[1].name,
        owner: entry[1].owner,
        type: jobType(entry[1].type),
        disabled: entry[1].disabled ? 'Disabled' : 'Enabled',
        runs: entry[1].stats ? entry[1].stats.length : 0,
        done: entry[1].is_done,
      }
      return acc+tmpl(record);
    }, "");
    return props;
  })
});

var jobDetail = new Reef('#jobDetailModal', {
  store: store,
  attachTo: app,
  template: function() {/*
    <div class="modal ${active}">
      <div class="modal-background" onclick="store.do('clearJobDetail')"></div>
       <div class="modal-card">
        <header class="modal-card-head">
          <p class="modal-card-title">
            ${title}
            <span class="tag ${statusClass}">${statusText}</span>
          </p>
          <button class="delete" aria-label="close" onclick="store.do('clearJobDetail')"></button>
        </header>
        <section class="modal-card-body">
          <pre>${prettyJob}</pre>
        </section>
        <footer class="modal-card-foot">
          <button class="button is-primary">Stats</button>
          <button class="button is-primary" onclick="toggleJobDisabled('${id}')">${changeStatusText}</button>
          <button class="button is-primary" onclick="runJob('${id}')">Run Manually</button>
          <button class="button is-danger" onclick="deleteJob('${id}')">Delete</button>
        </footer>
      </div>
      <button class="modal-close is-large" aria-label="close" onclick="store.do('clearJobDetail')"></button>
    </div>
  */}.extractComment().propsInjector(function(props){
    return !props.jobDetail ? props : Object.assign(props, {
      prettyJob: JSON.stringify(props.jobDetail, undefined, 4),
      title: props.jobDetail.name,
      active: 'is-active',
      statusClass: props.jobDetail.disabled ? 'is-warning' : 'is-success',
      statusText: props.jobDetail.disabled ? 'Disabled' : 'Enabled',
      changeStatusText: props.jobDetail.disabled ? 'Enable' : 'Disable',
      id: props.jobDetail.id,
    })
    // if (props.jobDetail) {
    //   props.prettyJob = JSON.stringify(props.jobDetail, undefined, 4);
    //   props.title = props.jobDetail.name;
    //   props.active = 'is-active';
    //   var disabled = props.jobDetail.disabled;
    //   props.statusClass = disabled ? 'is-warning' : 'is-success';
    //   props.statusText = disabled ? 'Disabled' : 'Enabled';
    //   props.changeStatusText = disabled ? 'Enable' : 'Disable';
    //   props.id = props.jobDetail.id;
    // }
    // return props;
  }),
})

function toggleJobDisabled(id) {
  var job = store.data.jobs[id];
  if (job) {
    job.disabled ? enableJob(job.id) : disableJob(job.id);
  }
}

function disableJob(id) {
  kala.disableJob(id).then(function(){
    store.do('setJobDisabled', id, true);
  });
}

function enableJob(id) {
  kala.enableJob(id).then(function(){
    store.do('setJobDisabled', id, false);
  }); 
}

function runJob(id) {
  kala.runJob(id)
    .then(function(){
      return kala.getJob(id);
    }).then(function(job){
      store.do('setJob', job);
    })
}

function deleteJob(id) {
  kala.deleteJob(id)
    .then(function(){
      store.do('deleteJob', id);
    });
}

function createJob(body) {
  kala.createJob(body)
    .then(function(id){
      return kala.getJob(id);
    }).then(function(job){
      store.do('setJob', job);
    });
}

kala.getJobs().then(function(resp){
  store.do('setJobs', resp.jobs);
});
