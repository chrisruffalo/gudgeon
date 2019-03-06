import React from 'react';
import Axios from 'axios';
import { ChartArea, ChartGroup, ChartLegend, ChartVoronoiContainer } from '@patternfly/react-charts';
import gudgeonStyles from '../../css/gudgeon-app.css';
import { css } from '@patternfly/react-styles';

export class QPSChart extends React.Component {
  containerRef = React.createRef();
  state = {
    width: 0,
    data: [],
    lastAtTime: null
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
          const newState = Object.assign({}, this.state, { data: newData, lastAtTime: newAtTime });
          this.setState(newState)
        }

        // set timeout and update data again
        setTimeout(() => {
          this.updateData()
        },10000);
      });
  }

  componentDidMount() {
    // resize
    setTimeout(() => {
      //this.setState({ width: this.containerRef.current.clientWidth, data: this.data });
      const newState = Object.assign({}, this.state, { width: this.containerRef.current.clientWidth });
      this.setState(newState)
      window.addEventListener('resize', this.handleResize);
    });

    // log
    console.log("mounted component...");

    // update data
    this.updateData();
  }

  componentWillUnmount() {
    window.removeEventListener('resize', this.handleResize);
  }

  getTooltipLabel = (datum) => {
    var queries = this.getDataItem(datum, 'gudgeon-total-interval-queries');
    var blocked = this.getDataItem(datum, 'gudgeon-blocked-interval-queries');
    if (blocked == 0 && queries == 0) {
      return null;
    }
    return `${this.prettyDate(datum.AtTime)}\nQueries : ${queries}\nBlocked: ${blocked}`;
  }

  handleResize = () => {
    this.setState({ width: this.containerRef.current.clientWidth });
  };

  getDataItem(dataItem, valueKey) {
    return dataItem != null && dataItem.Values != null && dataItem.Values[valueKey] != null ? dataItem.Values[valueKey].count : 0;
  };

  goDateOptions = {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    year: 'numeric', 
    month: '2-digit', 
    day: '2-digit',
    timeZoneName: "short"
  };

  prettyDate(date) {
    if ( date == null ) {
      return "";
    }

    return new Date(Date.parse(date)).toLocaleString(undefined, this.goDateOptions);
  }

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
