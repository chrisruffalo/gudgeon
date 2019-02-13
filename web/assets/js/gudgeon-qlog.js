new Vue({
  el: '#qlog',
  data () {
    return {
      headers: [
        { text: 'Client Address', value: 'Address' },
        { text: 'Domain', value: 'RequestDomain'},
        { text: 'Type', value: 'RequestType' },
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