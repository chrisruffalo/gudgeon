// percentage filter
Vue.filter('percentage', function (value, decimals) {
  if (!value) value = 0
  if (!decimals) decimals = 0

  value = value * 100
  return Math.round(value * Math.pow(10, decimals)) / Math.pow(10, decimals) + '%'
})

// display number as it would be in the current locale
Vue.filter('localeNumber', function (value) {
  return Number(value).toLocaleString()
})
