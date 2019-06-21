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

  return (new Date(Date.parse(date))).toLocaleString(undefined, goDateOptions);
}

export function HumanBytes(bytes) {
  return prettyBytes(bytes * 1);
}

export function LocaleInteger(number) {
  if ((number * 1) < 1) {
    return 0;
  }
  return LocaleNumber(number)
}

export function LocaleNumber(number) {
  return (number * 1).toLocaleString();
}

export function ProcessorPercentFormatter(value) {
  return LocaleNumber(value / 100) + "%";
};