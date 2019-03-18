import React from 'react';
import Axios from 'axios';
import { 
  FormSelect,
  FormSelectOption
} from '@patternfly/react-core';
import { PrettyDate } from './helpers.js';
import { ChartArea, ChartGroup, ChartLabel, ChartLegend, ChartVoronoiContainer, ChartTooltip, ChartAxis } from '@patternfly/react-charts';
import gudgeonStyles from '../../css/gudgeon-app.css';
import { css } from '@patternfly/react-styles';

export class QPSChart extends React.Component {
  constructor(props) {
    super(props);
  };

  containerRef = React.createRef();
  state = {
    width: 0,
    data: [],
    lastAtTime: null,
    interval: (60 * 30), // start at 30min
    timer: null,
    domainMaxY: 0 // calculate max Y
  };

  options = [
    { value: (60 * 30).toString(), label: '30m', disabled: false},
    { value: (60 * 60), label: '1h', disabled: false },
    { value: (60 * 60 * 2), label: '2h', disabled: false },
    { value: (60 * 60 * 4), label: '4h', disabled: false },
    { value: (60 * 60 * 6), label: '6h', disabled: false }
  ];

  onTimeIntervalChange = (value, event) => {
    // clear old timer
    var { timer } = this.state
    if ( timer != null ) {
      clearTimeout(timer)
    }

    // update table
    const interval = (Math.floor(Date.now()/1000) - value).toString();
    this.setState({ lastAtTime: interval, data: [], interval: value, timer: null }, this.updateData);
  };

  updateData() {
    // get from state but allow state to be reset to null without additional logic
    var { lastAtTime, interval } = this.state
    if ( null == lastAtTime || 100000 >= lastAtTime ) {
      lastAtTime = (Math.floor(Date.now()/1000) - interval).toString()
    }

    // clear any existing timers
    var { timer } = this.state
    if ( timer != null ) {
      clearTimeout(timer)
    }

    // basic queries start at the given time
    var params = {
      start: lastAtTime
    }

    // use this to filter query down to what is requested for the chart
    var metricsSelected = "";
    var idx = 0;
    for ( idx in this.props.metrics ) {
      if ( idx > 0 ) {
        metricsSelected = metricsSelected + ","
      }
      metricsSelected = metricsSelected + this.props.metrics[idx].key
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
          var { data } = this.state
          // concat query data
          var newData = [];
          // lowest time
          const minTime = (Math.floor(Date.now()/1000) - interval)

          if ( data != null && data.length > 0 ) {
            newData = data.filter( datapoint => Math.floor(new Date(datapoint.AtTime) / 1000) >= minTime)
            // add in new data
            newData = newData.concat(response.data)
          } else {
            newData = response.data
          }

          // find maxY
          var maxY = 0
          var idx;
          var k;
          for ( idx in newData ) {
            for ( k in newData[idx].Values) {
              if( (newData[idx].Values[k].count * 1.0) > maxY ) {
                maxY = (newData[idx].Values[k].count * 1.0);
              }
            }
          }

          // update at time
          var lastElement = response.data[response.data.length - 1];
          var newAtTime = (Math.floor(new Date(lastElement.AtTime) / 1000) + 1).toString(); // time is in ms we need in s

          // change state
          this.setState({ data: newData, lastAtTime: newAtTime, domainMaxY: maxY });
        }

        // set timeout and update data again
        var newTimer = setTimeout(() => {
          this.updateData()
        },10000);

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
    // (safely) load state
    var stateString = localStorage.getItem("gudgeon-" + this.props.stateid + "-state");
    if (stateString == "" || stateString == null) {
      stateString = "{}"
    }
    var savedState = JSON.parse(stateString);
    savedState['timer'] = null;

    // resize
    setTimeout(() => {
      if ( this.containerRef != null ) {
        this.setState({ width: this.containerRef.current.clientWidth })
      }
      window.addEventListener('resize', this.handleResize);
    });

    // load state and update data
    this.setState(savedState, this.updateData);
  }

  componentWillUnmount() {
    var { timer } = this.state
    if ( timer != null ) {
      clearTimeout(timer)
    }

    // remove listener
    window.removeEventListener('resize', this.handleResize);

    // save state as last action
    localStorage.setItem("gudgeon-" + this.props.stateid + "-state", JSON.stringify(this.state));
  }

  mapSingleValue(data, metric) {
    if ( data == null || data.length < 1 || data.map == null ) {
      return [];
    }

    var output = data.map(d => {
      return {
        "AtTime": d.AtTime,
        "FromTime": d.FromTime,
        "Value": metric != null && metric.key != null && d != null && d.Values != null && d.Values[metric.key] != null ? d.Values[metric.key].count : 0,
        "Metric": metric
      };
    });
    return output;
  }

  getTooltipLabel = (datum) => {
    if ( datum == null || datum.Value == null ) {
      return null
    }

    var value = datum.Value != null ? datum.Value : 0;
    if ( this.props.formatter != null ) {
      value = this.props.formatter(value)
    }
    if ( datum != null && datum.Metric != null && datum.Metric.name != null ) {
      value = datum.Metric.name + " : " + value
    }
    return value
  }

  handleResize = () => {
    this.setState({ width: this.containerRef.current.clientWidth });
  };

  getDataItem(dataItem, valueKey) {
    return dataItem != null && dataItem.Value != null ? dataItem.Value : 0;
  };

  render() {
    const { width, data, domainMaxY } = this.state;
    const container = <ChartVoronoiContainer labels={ this.getTooltipLabel } labelComponent={ <ChartTooltip/> } />;

    const rows = []
    this.props.metrics.forEach( metric => {
      rows.push(
        <ChartArea key={metric.key} samples={10} domain={{ y: [0, domainMaxY * 1.25] }} scale={{ x: "time", y: "linear" }} data={ this.mapSingleValue(data, metric) } x={ (d) => d.AtTime } y={ (d) => this.getDataItem(d) } />
      )
    });

    const legend = this.props.metrics.map( metric => { return { "name": metric.name } });

    return (
      <React.Fragment>
        <div>
          <FormSelect value={this.state.interval} onChange={this.onTimeIntervalChange} aria-label="FormSelect Input">
            {this.options.map((option, index) => (
              <FormSelectOption isDisabled={option.disabled} key={index} value={option.value} label={option.label} />
            ))}
          </FormSelect>
        </div>               
        <div ref={this.containerRef}>
          <div className="chart-overflow">
            <ChartGroup containerComponent={container} height={200} width={width}>
              { rows }
            </ChartGroup>
          </div>
          <ChartLegend
            data={ legend }
            height={50}
            width={width}
          />        
        </div>
      </React.Fragment>
    );
  }
}
