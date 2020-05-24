(function(window){

	var KalaHost = window.location.protocol + "//" + window.location.hostname + (window.location.port ? ':'+window.location.port : '');
	var Version = 'v1';
	var ApiBase = KalaHost + "/api/" + Version + "/";
	var ApiJobEndpoint = ApiBase + "job/";

	var service = {
		getJobs: function() {
			return fetch(ApiJobEndpoint)
				.then(function(response){
					return response.json()
				}).catch(function(ex){
					console.error('parsing jobs failed: ', ex)
				});
		},
		getJob: function(id) {
			return fetch(ApiJobEndpoint + id + '/')
				.then(function(response){
					return response.json()
				}).catch(function(ex){
					console.error('parsing job failed: ', ex)
				});
		},
		disableJob: function(id) {
			return fetch(ApiJobEndpoint + 'disable/' + id + '/', {method: 'post'})
				.catch(function(ex){
					console.error('disabling job failed: ', ex)
				})
		},
		enableJob: function(id) {
			return fetch(ApiJobEndpoint + 'enable/' + id + '/', {method: 'post'})
				.catch(function(ex){
					console.error('enabling job failed: ', ex)
				})
		},
		runJob: function(id) {
			return fetch(ApiJobEndpoint + 'start/' + id + '/', {method: 'post'})
				.catch(function(ex){
					console.error('starting job failed: ', ex)
				})
		},
		deleteJob: function(id) {
			return fetch(ApiJobEndpoint + id + '/', {method: 'delete'})
				.catch(function(ex){
					console.error('deleting job failed: ', ex)
				})
		},
		createJob: function(job) {
			return fetch(ApiJobEndpoint, {method: 'post', body: JSON.stringify(job)})
				.then(function(response){
					return response.json()
				})
				.then(function(json){
					return json.id;
				})
				.catch(function(ex){
					console.error('creating job failed: ', ex)
				})
		},
		jobStats: function(id) {
			return fetch(ApiJobEndpoint + 'stats/' + id + '/')
				.catch(function(ex){
					console.error('getting job stats failed: ', ex)
				})
		},
		metrics: function(id) {
			return fetch(ApiBase + 'stats/')
				.catch(function(ex){
					console.error('getting metrics failed: ', ex)
				})
		},
	}

	this.kala = service;

})(this);
