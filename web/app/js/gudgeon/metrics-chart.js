import React from 'react';
import Axios from 'axios';
import '../../css/gudgeon-chart.css';
import c3 from 'c3';
import 'c3/c3.css';
import { 
  Form,
  FormSelect,
  FormSelectOption
} from '@patternfly/react-core';
import {HumanBytes, LocaleNumber, ProcessorPercentFormatter} from "./helpers";

const CHART_REFRESH_TIMEOUT = 2500;

export class Metrics {}
Metrics.Queries = {
  label: "Queries",
  formatter: LocaleNumber,
  series: {
    queries: { name: "Queries/s", key: "gudgeon-session-queries-ps" },
    blocked: { name: "Blocked/s", key: "gudgeon-session-blocks-ps" } ,
    latency: { name: "Service Time (ms)", key: "gudgeon-query-time", axis: "y2", use_average: true }
  }
};
Metrics.Memory = {
  label: "Memory",
  formatter: HumanBytes,
  series: {
    heap: { name: "Allocated Heap", key: "gudgeon-allocated-bytes" },
    rss: { name: "Resident Memory", key: "gudgeon-process-used-bytes" },
    cache: { name: "Cache Entries", key: "gudgeon-cache-entries", axis: "y2" }
  }
};
Metrics.Threads = {
  label: "Threads",
  formatter: LocaleNumber,
  series: {
    threads: { name: "Threads", key: "gudgeon-process-threads" },
    routines: { name: "Go Routines", key: "gudgeon-goroutines" }
  }
};
Metrics.CPU = {
  label: "CPU",
  formatter: ProcessorPercentFormatter,
  domain: {
    maxY: 10000, // processor use is in 1000ths of a percent
    minY: 0
  },
  ticks: [5000, 10000],
  series: {
    cpu: { name: "CPU Use", key: "gudgeon-cpu-hundreds-percent" }
  }
};

export class GudgeonChart extends React.Component {
  constructor(props) {
    super(props);

    // start with null chart
    this.chart = null;

    // timer for queries
    this.timer = null;

    // state doesn't need to change every time the chart is rendered
    this.columns = [];

    // create a new chart id for use in div name
    if(props.chartName !== null && props.chartName.length > 0) {
      this.chartId = "gudgeon-chart-id-" + props.chartName;
    } else {
      this.chartId = "gudgeon-chart-id";
    }
  };

  state = {
    width: 0,
    lastAtTime: null,
    interval: (60 * 30), // start at 30min
    domainMaxY: 0, // calculate max Y
    selected: Object.keys(this.props.metrics)[0]
  };

  // these colors are from the patternfly pallete
  colors = [
    '#004368',
    '#00659c',
    '#0088ce',
    '#39a5dc',
    '#7dc3e8'
  ];

  options = [
    { value: (60 * 10), label: '10m', disabled: false},
    { value: (60 * 30), label: '30m', disabled: false},
    { value: (60 * 60), label: '1h', disabled: false },
    { value: (60 * 60 * 2), label: '2h', disabled: false },
    { value: (60 * 60 * 6), label: '6h', disabled: false },
    { value: (60 * 60 * 12), label: '12h', disabled: false },
    { value: (60 * 60 * 24), label: '24h', disabled: false },
    { value: -1, label: 'All Time', disabled: false },
  ];

  onMetricKeyChange = (value, event) => {
    // clear old timer
    if ( this.timer != null ) {
      clearTimeout(this.timer)
    }

    // destroy old chart
    if ( this.chart != null ) {
      this.chart.destroy();
      this.chart = null;
    }

    // nuke data
    this.columns = [];

    this.setState({ lastAtTime: null, selected: value }, this.updateData );
  };

  onTimeIntervalChange = (value, event) => {
    // clear old timer
    if ( this.timer != null ) {
      clearTimeout(this.timer)
    }

    // keep settings, reload chart {
    if ( this.chart != null ) {
      this.chart.destroy();
      this.chart = null;
    }

    // nuke data
    this.columns = [];

    // update table
    this.setState({ lastAtTime: null, interval: value }, this.updateData );
  };

  updateData() {
    let { lastAtTime, interval, selected } = this.state;

    // get from state but allow state to be reset to null without additional logic
    // and also make sure that the last at time is after the currently selected interval
    if ( null == lastAtTime ) {
      if ( interval < 1 ) {
          lastAtTime = (new Date(0) / 1000).toString();
        } else {
          lastAtTime = (Math.floor(Date.now()/1000) - interval).toString()
        }      
    }

    // clear any existing timers
    if ( this.timer != null ) {
      clearTimeout(this.timer);
    }

    // basic queries start at the given time as long as the interval is positive
    let params = {
      start: lastAtTime,
    };

    Axios
      .get("/api/metrics/query", {
        params: params
      })
      .then(response => {
        if ( response == null || response.data == null || response.data.length < 0 ) {
          // set quicker timeout and quit processing response
          this.timer = setTimeout(() => { this.updateData() },CHART_REFRESH_TIMEOUT / 2);
          return;
        }

        // measure domain max
        let newDomainMaxY = 0;

        // set x column
        if (this.columns.length < 1 || this.columns[0].length < 1) {
          this.columns[0] = ['x'];
        }

        // split data into columns
        response.data.forEach((item) => {
          if ( !item || !item.Values ) {
            return;
          }

          // the index starts at one
          let idx = 1;
          for (let key in this.props.metrics[selected].series ) {
            // guard series
            if (!this.props.metrics[selected].series.hasOwnProperty(key)) {
              continue
            }

            // for each metric we need to grab values
            let metric = this.props.metrics[selected].series[key];

            // start series if series isn't started
            if ( this.columns[idx] == null || this.columns[idx].length < 0 ) {
              this.columns[idx] = [metric.name];
            }

            // add x values first time
            if ( idx < 2 ) {
              this.columns[0].push(new Date(item.AtTime));
            }

            // determine if we should use the average instead of the count (default false)
            let use_average = metric.use_average;

            // push new item into value array for that key and push a 0 if there is no value (makes missing metrics a "0" until they start)
            if (!use_average && item.Values[metric.key] != null && item.Values[metric.key].count != null) {
              this.columns[idx].push(item.Values[metric.key].count);
            } else if(use_average && item.Values[metric.key] != null && item.Values[metric.key].average != null) {
              this.columns[idx].push(item.Values[metric.key].average);
            } else {
              this.columns[idx].push(0);
            }

            // move to next index for the next data series
            idx++;
          }
        });

        // cull old data as needed
        if ( interval > 0 ) {
          // lowest time possible
          const minTime = (Math.floor(Date.now()/1000) - interval);

          // while there are date elements and the current date element
          // (first non 'x' element) is less than the minimum time keep
          // slicing out that first element in all the lists
          while ( this.columns[0].length > 1 && Math.floor((this.columns[0][1] * 1) / 1000) < minTime ) {
            let idx = 0;
            for ( idx in this.columns ) {
              // pop off the SECOND element (the FIRST is the column name, maybe there's a better way to do this)
              this.columns[idx].splice(1, 1);
            }
          }
        }

        // determine domain max after culling
        for ( let idx = 1; idx < this.columns.length; idx++ ) {
          let series_name = this.columns[idx][0];
          let series = null;
          // get series from name ... there has to be a better way to find what axis the series is on
          for ( key in this.props.metrics[selected].series ) {
            if (!this.props.metrics[selected].series.hasOwnProperty(key)) {
              continue;
            }

            if ( series_name === this.props.metrics[selected].series[key]["name"] ) {
              series = this.props.metrics[selected].series[key]
            }
          }

          // if the series is available and is on y2 then don't use it for the potential domain max
          if ( series != null && (series.hasOwnProperty("axis") && series["axis"] === "y2") ) {
            continue
          }

          for ( let jdx = 1; jdx < this.columns[idx].length; jdx++ ) {
            let item = this.columns[idx][jdx];
            if ( !isNaN(item) && item >= newDomainMaxY ) {
              newDomainMaxY = item;
            }
          }
        }

        // update at time
        let lastElement = response.data[response.data.length - 1];
        let newAtTime = (Math.floor(new Date(lastElement.AtTime) / 1000) + 1).toString(); // time is in ms we need in s

        // change state and callback
        this.setState({ lastAtTime: newAtTime, domainMaxY: newDomainMaxY }, this.handleChart);

        // set timeout and update data again
        this.timer = setTimeout(() => { this.updateData() },CHART_REFRESH_TIMEOUT);
      }).catch((error) => {
        this.timer = setTimeout(() => { this.updateData() },CHART_REFRESH_TIMEOUT * 4); // on error try again after waiting 4x interval
      });
  }

  componentDidMount() {
    // start update cycle
    this.updateData();

    setTimeout(() => {
      window.addEventListener('resize', this.handleResize);
    });
  }

  componentWillUnmount() {
    if ( this.timer != null ) {
      clearTimeout(this.timer)
    }

    window.removeEventListener('resize', this.handleResize);

    if ( this.chart != null ) {
      this.chart.destroy();
    }
  }

  // create a format function that wraps the axis formatter
  wrapAxisFormatter(inputFormatter) {
    return (value) => { 
      // no negative values
      if ( (value + "").indexOf("-") === 0 ) {
        return "";
      }

      // floor value if not otherwise formatted
      if (!isNaN(value)) {
        value = Math.floor(value);
      }

      // use the provided formatter
      if ( inputFormatter != null ) {
        value = inputFormatter(value);

        return value !== "0" ? value : "";
      }

      // don't return a ones place value on non-formatted ticks,
      // so that 152 should return 160
      return value > 0 ? Math.ceil(value / 10) * 10 : "";
    };
  }

  generateChart() {
    const { selected, domainMaxY } = this.state;

    // can't layout a chart/metric that is missing. this happens
    // when a loaded state doesn't exist anymore
    if ( this.props.metrics[selected] == null ) {
        return;
    }

    const chartSettings = {
      bindto: "#" + this.chartId,
      padding: { 
        top: 15, 
        bottom: 10
      },
      point: {
        show: false
      },
      data: {
        x: 'x',
        columns: this.columns,
        type: 'area'
      },
      axis: {
          y: {
            min: 0,
            padding: { 
              top: 0,
              bottom: 10
            },            
            tick: {
              outer: false,
              format: this.wrapAxisFormatter(this.props.metrics[selected].formatter),
              count: 3,
              culling: {
                max: 3
              }
            }
          },
          x: {
              type: 'timeseries',
              tick: {
                outer: false,
                count: 3,
                multiline: true,
                culling: {
                  max: 5
                },
                format: '%I:%M%p %m/%d/%y'
              }
          },
          y2: {
            show: false,
            min: 0,
            padding: { 
              top: 0,
              bottom: 10
            },             
            tick: {
              outer: false,
              format: this.wrapAxisFormatter(null),
              count: 2,
              culling: {
                max: 3
              }
            }
          }
      },
      tooltip: { 
        format: { 
          // real bad hack for y2 values to not use the same formatter
          value: (value, ratio, id) => {
            let key;
            for ( key in this.props.metrics[selected].series ) {
              if ( id === this.props.metrics[selected].series[key].name && this.props.metrics[selected].series[key].axis === "y2" ) {
                return value;
              }
            }
            return this.props.metrics[selected].formatter(value);
          }
        } 
      }     
    };

    // determine axes
    let axes = {};
    for ( let key in this.props.metrics[selected].series ) {
      // guard property
      if(!this.props.metrics[selected].series.hasOwnProperty(key)) {
        continue;
      }

      let yaxis = this.props.metrics[selected].series[key].axis;
      if ( yaxis !== "y2" ) {
        yaxis = "y";
      } else {
        chartSettings['axis']['y2']['show'] = true;
        chartSettings['axis']['y2']['min'] = 0;
      }
      axes[this.props.metrics[selected].series[key].name] = yaxis;
    }
    chartSettings['data']['axes'] = axes;

    // set colors from color map
    chartSettings['data']['colors'] = [];
    for ( let idx = 0; idx < this.columns.length; idx ++) {
      chartSettings['data']['colors'][this.columns[idx][0]] = this.colors[idx - 1];
    }

    // if no domain is specified, use calculated domain
    if ( this.props.metrics[selected].domain != null && this.props.metrics[selected].domain.maxY != null ) {
      chartSettings['axis']['y']['max'] = this.props.metrics[selected].domain.maxY;
    } else {
      if ( domainMaxY > 0 ) {
        let calcDomainMax = Math.floor(domainMaxY * 1.25);
        if ( calcDomainMax < 10 ) {
          calcDomainMax = 10;
        }        
        chartSettings['axis']['y']['max'] = calcDomainMax; 
      }
    }

    if ( this.props.metrics[selected].ticks ) {
      chartSettings['axis']['y']['tick']['values'] = this.props.metrics[selected].ticks;
    }

    // generate the chart (bound to that div)
    return c3.generate(chartSettings);
  }

  handleChart() {
    // if the chart hasn't been built yet then build it before loading the data
    if ( this.chart == null ) {
      this.chart = this.generateChart();
    } else {
      const { selected, domainMaxY } = this.state;

      // load chart using the object directly
      this.chart.load({ columns: this.columns });

      // manually set domain max if needed
      if ( this.props.metrics[selected]['domain'] == null || this.props.metrics[selected]['domain']['maxY'] == null ) {
        let calcDomainMax = Math.floor(domainMaxY * 1.25);
        if ( calcDomainMax < 10 ) {
          calcDomainMax = 10;
        }
        this.chart.axis.max({ y: calcDomainMax });
      }
    }
  }

  render() {
    const metricOptions = [];
    if ( this.props.metrics != null && Object.keys(this.props.metrics).length > 1) {
      let key = null;
      for ( key in this.props.metrics) {

        metricOptions.push(
          <FormSelectOption isDisabled={false} key={key} value={key} label={this.props.metrics[key].label} />
        );
      }
    }

    let metricSelect = null;
    if ( metricOptions.length > 1 ) {
      metricSelect = (
        <FormSelect style={{ "gridColumnStart": 1 }} value={this.state.selected} onChange={this.onMetricKeyChange} aria-label="Select Metric">
          { metricOptions }
        </FormSelect>
      );
    }

    return (
      <React.Fragment>
        <div>
          <Form isHorizontal>
            { metricOptions.length > 1 ? metricSelect : null }
            <FormSelect style={{ "gridColumnStart": metricOptions.length > 1 ? 2 : 1 }} value={this.state.interval} onChange={this.onTimeIntervalChange} aria-label="Select Time Interval">
              {this.options.map((option, index) => (
                <FormSelectOption isDisabled={option.disabled} key={index} value={option.value} label={option.label} />
              ))}
            </FormSelect>
          </Form>
        </div>               
        <div id={ this.chartId }/>
      </React.Fragment>
    );
  }
}