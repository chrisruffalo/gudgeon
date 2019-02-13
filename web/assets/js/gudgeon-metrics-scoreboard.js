// web ui logic entrypoint
var app = new Vue({
  el: '#metrics-scoreboard',
  data: {
    metrics: {
      'gudgeon-total-rules': { 'count': 0 },
      'gudgeon-total-queries': { 'count': 0 },
      'gudgeon-blocked-queries': { 'count': 0 },
    },
    retryIntervals: {},
  },
  methods: {
    fetchMetric: function(retryInterval) {
      axios
        .get('/api/metrics')
        .then(response => {
          this['metrics'] = response.data
          if (retryInterval > 0) {
            this.retryIntervals['metrics'] = window.setTimeout(function() {
              app.fetchMetric(retryInterval)
            }
            , retryInterval)
          }
        })
        .catch(error => {
          console.log(error)
        })
    }
  },
  mounted () {
    this.fetchMetric(1500)
  },
  beforeDestroy() {
    for(var key in this.retryIntervals) {
        window.clearInterval(this.retryIntervals[key])
    }
  }
})