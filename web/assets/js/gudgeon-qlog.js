new Vue({
  el: '#qlog',
  data () {
    return {
      pagination: {
        sortBy: 'Created',
        descending: true,
      },      
      headers: [
        { text: 'Client Address', value: 'Address' },
        { text: 'Blocked', value: 'Blocked'},
        { text: 'Question Domain', value: 'RequestDomain'},
        { text: 'Question Type', value: 'RequestType' },
        { text: 'Response', value: 'ResponseText'},
        { text: 'Time', value: 'Created', align: "right"}
      ],
      data: [],
    }
  },
  methods: {
    fetchLogs: function() {
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
    this.fetchLogs()
  },
})