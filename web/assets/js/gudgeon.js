// web ui logic entrypoint
var app = new Vue({
  el: '#main',
  data: {
    total_queries: { "count": 0 },
    blocked_queries: { "count": 0 },
    total_rules: { "count": 0 },
    retryIntervals: {}
  },
  methods: {
    fetchMetric: function(metric_type, metric_name, key, retryInterval) {
      axios
        .get('/web/api/metrics/' + metric_type + '/' + metric_name)
        .then(response => {
          this[key] = response.data
          if (retryInterval > 0) {
            this.retryIntervals[key] = window.setTimeout(function() {
              app.fetchMetric(metric_type, metric_name, key, retryInterval)
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
    this.fetchMetric("counter","gudgeon-total-rules","total_rules", 60000)
    this.fetchMetric("meter","gudgeon-total-queries","total_queries", 750)
    this.fetchMetric("meter","gudgeon-blocked-queries","blocked_queries", 750)
  },
  beforeDestroy() {
    for(var key in this.retryIntervals) {
        window.clearInterval(this.retryIntervals[key])
    }
  }
})