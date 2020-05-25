var router = new Reef.Router({
  title: '{{title}} | Kala Web Interface',
  routes: [
    {
      id: 'jobs',
      title: 'Jobs',
      url: '/webui/',
    },
    {
      id: 'jobs',
      title: 'Jobs',
      url: '/jobs/'
    },
    {
      id: 'metrics',
      title: 'Metrics',
      url: '/metrics/'
    }
  ],
  useHash: true,
});
