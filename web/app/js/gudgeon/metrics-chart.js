import React from 'react';
import Axios from 'axios';
import { PrettyDate } from './helpers.js';
import { ChartArea, ChartGroup, ChartLegend, ChartVoronoiContainer } from '@patternfly/react-charts';
import gudgeonStyles from '../../css/gudgeon-app.css';
import { css } from '@patternfly/react-styles';

export class QPSChart extends React.Component {
  containerRef = React.createRef();
  state = {
    width: 0,
    data: [],
    lastAtTime: null,
    timer: null
  };

  updateData() {
    // get from state but allow state to be reset to null without additional logic
    var { lastAtTime } = this.state
    if ( null == lastAtTime ) {
      lastAtTime = (Math.floor(Date.now()/1000) - (60 * 60)).toString()
    }

    Axios
      .get("api/metrics/query", {
        params: {
          // one hour ago
          start: lastAtTime,
        }
      })
      .then(response => {
        if ( response != null && response.data != null && response.data.length > 0 ) {
          var { data } = this.state
          // concat query data
          var newData = data.concat(response.data)
          // update at time
          var lastElement = response.data[response.data.length - 1];
          var newAtTime = (Math.floor(new Date(lastElement.AtTime) / 1000) + 1).toString() // time is in ms we need in s
          // change state
          this.setState({ data: newData, lastAtTime: newAtTime })
        }

        // set timeout and update data again
        var newTimer = setTimeout(() => {
          this.updateData()
        },10000);

        // update the data in the state
        this.setState({ timer: newTimer })        
      });
  }

  componentDidMount() {
    // resize
    setTimeout(() => {
      this.setState({ width: this.containerRef.current.clientWidth })
      window.addEventListener('resize', this.handleResize);
    });

    // update data
    this.updateData();
  }

  componentWillUnmount() {
    var { timer } = this.state
    if ( timer != null ) {
      clearTimeout(timer)
      this.setState({ timer: null })
    }

    window.removeEventListener('resize', this.handleResize);
  }

  getTooltipLabel = (datum) => {
    var queries = this.getDataItem(datum, 'gudgeon-total-interval-queries');
    var blocked = this.getDataItem(datum, 'gudgeon-blocked-interval-queries');
    if (blocked == 0 && queries == 0) {
      return null;
    }
    return `${PrettyDate(datum.AtTime)}\nQueries : ${queries}\nBlocked: ${blocked}`;
  }

  handleResize = () => {
    this.setState({ width: this.containerRef.current.clientWidth });
  };

  getDataItem(dataItem, valueKey) {
    return dataItem != null && dataItem.Values != null && dataItem.Values[valueKey] != null ? dataItem.Values[valueKey].count : 0;
  };

  render() {
    const { width } = this.state;
    const { data } = this.state;
    const container = <ChartVoronoiContainer labels={this.getTooltipLabel} />;

    return (
      <div ref={this.containerRef} className={css(gudgeonStyles.maxHeight)}>
        <div className="chart-overflow">
          <ChartGroup containerComponent={container} height={200} width={width}>
            <ChartArea scale={{ x: "time", y: "linear" }} data={data} x={ (d) => d.AtTime } y={ (d) => this.getDataItem(d, 'gudgeon-total-interval-queries') } />
            <ChartArea scale={{ x: "time", y: "linear" }} data={data} x={ (d) => d.AtTime } y={ (d) => this.getDataItem(d, 'gudgeon-blocked-interval-queries') } />
          </ChartGroup>
        </div>
        <ChartLegend
          data={[{ name: 'Queries' }, { name: 'Blocked' }]}
          height={50}
          width={width}
        />
      </div>
    );
  }
}
