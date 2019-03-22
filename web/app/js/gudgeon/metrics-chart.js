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
import { PrettyDate } from './helpers.js';
import gudgeonStyles from '../../css/gudgeon-app.css';
import { css } from '@patternfly/react-styles';

export class GudgeonChart extends React.Component {
  constructor(props) {
    super(props);

    // start with null chart
    this.chart = null;

    // create a new chart ref
    this.chartId = "gudgeon-chart-id" + props.chartName;
  };

  state = {
    width: 0,
    columns: [],
    lastAtTime: null,
    interval: (60 * 30), // start at 30min
    timer: null,
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
    { value: (60 * 5), label: '5m', disabled: false},
    { value: (60 * 10), label: '10m', disabled: false},
    { value: (60 * 30), label: '30m', disabled: false},
    { value: (60 * 60), label: '1h', disabled: false },
    { value: (60 * 60 * 2), label: '2h', disabled: false },
    { value: (60 * 60 * 4), label: '4h', disabled: false },
    { value: (60 * 60 * 6), label: '6h', disabled: false },
    { value: (60 * 60 * 6), label: '12h', disabled: false },
    { value: (60 * 60 * 6), label: '24h', disabled: false },
    { value: -1, label: 'All Time', disabled: false },
  ];

  onMetricKeyChange = (value, event) => {
    // clear old timer
    var { timer } = this.state
    if ( timer != null ) {
      clearTimeout(timer)
    }

    // destroy old chart
    if ( this.chart != null ) {
      this.chart.destroy();
      this.chart = null;
    }

    this.setState({ lastAtTime: null, columns: [], timer: null, selected: value }, this.updateData );
  };

  onTimeIntervalChange = (value, event) => {
    // clear old timer
    var { timer } = this.state
    if ( timer != null ) {
      clearTimeout(timer)
    }

    // keep settings, reload chart {
    if ( this.chart != null ) {
      this.chart.destroy();
      this.chart = null;
    }

    // update table
    this.setState({ lastAtTime: null, columns: [], interval: value, timer: null }, this.updateData );
  };

  updateData() {
    var { lastAtTime, interval, selected } = this.state

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
    var { timer } = this.state
    if ( timer != null ) {
      clearTimeout(timer)
    }

    // basic queries start at the given time as long as the interval is positive
    var params = { 
      start: lastAtTime,
      condense: "true" 
    };

    // use this to filter query down to what is requested for the chart
    var metricsSelected = "";
    var key;
    var idx = 0;
    for ( key in this.props.metrics[selected].series ) {
      if ( idx > 0 ) {
        metricsSelected = metricsSelected + ","
      }
      metricsSelected = metricsSelected + this.props.metrics[selected].series[key].key
      idx++;
    }
    if ( metricsSelected != "" || metricsSelected.length > 0 ) {
      params["metrics"] = metricsSelected
    }

    Axios
      .get("/api/metrics/query", {
        params: params
      })
      .then(response => {
        if ( response != null && response.data != null && response.data.length > 0 ) {
          var { columns } = this.state

          // concat query data
          var newData = columns;

          // measure domain max
          var newDomainMaxY = 0;

          // set x column
          if (columns.length < 1 || columns[0].length < 1) {
            columns[0] = ['x'];
          }

          // split data into columns
          response.data.forEach((item) => {
            if ( !item || !item.Values ) {
              return;
            }

            // the index starts at one
            var idx = 1;
            for ( key in this.props.metrics[selected].series ) {
              // for each metric we need to grab values
              var metric = this.props.metrics[selected].series[key]

              // start series if series isn't started
              if ( newData[idx] == null || newData[idx].length < 0 ) {
                newData[idx] = [ metric.name ];
              }

              // add x values first time
              if ( idx < 2 ) {
                newData[0].push(new Date(item.AtTime));
              }

              // push new item into value array for that key and push a 0 if there is no value (makes missing metrics a "0" until they start)
              const val = item.Values[metric.key] != null && item.Values[metric.key].count != null ? item.Values[metric.key].count : 0
              newData[idx].push(val);

              // move to next index for the next data series
              idx++;
            }
          });

          // cull old data as needed
          if ( interval > 0 ) { 
            // lowest time possible
            const minTime = (Math.floor(Date.now()/1000) - interval)

            // while there are date elements and the current date element 
            // (first non 'x' element) is less than the minimum time keep
            // slicing out that first element in all the lists
            while ( newData[0].length > 1 && Math.floor((newData[0][1] * 1) / 1000) < minTime ) {
              var idx = 0;
              for ( idx in newData ) {
                // pop off the SECOND element (the FIRST is the column name, maybe there's a better way to do this)
                newData[idx].splice(1, 1);
              }
            }
          }

          // determine domain max after culling
          for ( var idx = 1; idx < newData.length; idx++ ) {
            for ( var jdx = 1; jdx < newData[idx].length; jdx++ ) {
              var item = newData[idx][jdx];
              if ( !isNaN(item) && item >= newDomainMaxY ) {
                newDomainMaxY = item;
              }
            }
          }

          // update at time
          var lastElement = response.data[response.data.length - 1];
          var newAtTime = (Math.floor(new Date(lastElement.AtTime) / 1000) + 1).toString(); // time is in ms we need in s

          // change state and callback
          this.setState({ columns: newData, lastAtTime: newAtTime, domainMaxY: newDomainMaxY }, this.handleChart);
        }

        // set timeout and update data again
        var newTimer = setTimeout(() => {
          this.updateData()
        },7000);

        // update the data in the state
        this.setState({ timer: newTimer })        
      }).catch((error) => {
        var newTimer = setTimeout(() => {
          this.updateData()
        },15000); // on error try again in 15s 

        // update the data in the state
        this.setState({ timer: newTimer })
      });
  }

  componentDidMount() {
    this.updateData();
  }

  componentWillUnmount() {
    var { timer } = this.state
    if ( timer != null ) {
      clearTimeout(timer)
    }

    if ( this.chart != null ) {
      this.chart.destroy();
    }
  }

  // create a format function that wraps the axis formatter
  wrapAxisFormatter(inputFormatter) {
    return (value) => { 
      // no negative values
      if ( (value + "").indexOf("-") == 0 ) {
        return "";
      }

      // floor value if not otherwise formatted
      if (!isNaN(value)) {
        value = Math.floor(value);
      }

      if ( inputFormatter != null ) {
        value = inputFormatter(value);

        return value != "0" ? value : "";
      }

      return value > 0 ? value : "";
    };
  }

  generateChart() {
    const { columns, selected, domainMaxY } = this.state;

    // can't layout a chart/metric that is missing. this happens
    // when a loaded state doesn't exist anymore
    if ( this.props.metrics[selected] == null ) {
        return;
    }

    var chartSettings = {
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
        columns: columns,
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
          }
      },
      tooltip: { 
        format: { 
          value: this.props.metrics[selected].formatter
        } 
      }      
    };

    // set colors from color map
    chartSettings['data']['colors'] = [];
    for ( var idx = 0; idx < columns.length; idx ++) {
      chartSettings['data']['colors'][columns[idx][0]] = this.colors[idx - 1];
    }

    // if no domain is specified, use calculated domain
    if ( this.props.metrics[selected].domain != null ) {
      if ( this.props.metrics[selected].domain.maxY != null ) {
        chartSettings['axis']['y']['max'] = this.props.metrics[selected].domain.maxY;
      }
    } else {
      if ( domainMaxY > 0 ) {
       chartSettings['axis']['y']['max'] = Math.floor(domainMaxY * 1.25); 
      }
    }

    if ( this.props.metrics[selected].ticks ) {
      chartSettings['axis']['y']['tick']['values'] = this.props.metrics[selected].ticks;
    }

    // generate the chart (bound to that div)
    return c3.generate(chartSettings);
  }

  handleChart() {
    if ( this.chart == null ) {
      this.chart = this.generateChart();
    } else {
      const { columns, selected, domainMaxY } = this.state;

      const yAxis = {
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
      }

      // change domain axis
      if ( domainMaxY > 0 ) {
        yAxis['max'] = Math.floor(domainMaxY * 1.25); 
      }

      this.chart.load({ columns: columns, axes: { y: yAxis } });
    }
  }

  render() {
    const metricOptions = [];
    if ( this.props.metrics != null && Object.keys(this.props.metrics).length > 1) {
      var key = null;
      for ( key in this.props.metrics) {

        metricOptions.push(
          <FormSelectOption isDisabled={false} key={key} value={key} label={this.props.metrics[key].label} />
        );
      }
    }

    var metricSelect = null
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
        <div id={ this.chartId } ></div>
      </React.Fragment>
    );
  }
}
