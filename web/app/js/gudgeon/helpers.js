const prettyBytes = require('pretty-bytes');

var goDateOptions = {
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  year: 'numeric', 
  month: '2-digit', 
  day: '2-digit',
  timeZoneName: "short"
};

export function PrettyDate(date) {
  if ( date == null ) {
    return "";
  }

  return new Date(Date.parse(date)).toLocaleString(undefined, goDateOptions);
}

export function HumanBytes(bytes) {
  return prettyBytes(bytes * 1);
}

export function LocaleNumber(number) {
  return (number * 1).toLocaleString();
}