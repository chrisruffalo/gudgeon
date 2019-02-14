new Vue({
  el: '#qlog',
  data () {
    return {
      headers: [
        { text: 'Client Address', value: 'Address' },
        { text: 'Question Domain', value: 'RequestDomain'},
        { text: 'Question Type', value: 'RequestType' },
        { text: 'Time', value: 'Created', align: "right"}
      ],
      data: [],
    }
  },
  methods: {
    fetchLogs: function(retryInterval) {
          axios
            .get('/api/log', {
              params: {
                limit: 'none',
                // one hour ago
                after: (Math.floor(Date.now()/1000) - (60 * 60)).toString(),
              }
            })
            .then(response => {
              this["data"] = response.data
            })
            .catch(error => {
              console.log(error)
            })
        }
  },
  mounted () {
    this.fetchLogs(1500)
  },
})