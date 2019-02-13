// percentage filter
Vue.filter('percentage', function (value, decimals) {
  if (!value) value = 0;
  if (!decimals) decimals = 0;

  value = value * 100
  return Math.round(value * Math.pow(10, decimals)) / Math.pow(10, decimals) + '%';
})

// display number as it would be in the current locale
Vue.filter('localeNumber', function (value) {
  return Number(value).toLocaleString();
})

// set options once
var goDateOptions = {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    year: 'numeric', 
    month: '2-digit', 
    day: '2-digit',
    timeZoneName: "short"
};
Vue.filter('godate', function (value) {
    // convert to local time
    return new Date(Date.parse(value)).toLocaleString(undefined, goDateOptions);
})
