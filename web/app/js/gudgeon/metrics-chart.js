import React from 'react';
import Axios from 'axios';
import { ChartArea, ChartGroup, ChartLegend, ChartVoronoiContainer } from '@patternfly/react-charts';

export class QPSChart extends React.Component {
  containerRef = React.createRef();
  state = {
    width: 0,
    data: null
  };

  updateData() {
    Axios
      .get("api/metrics/query", {
        params: {
          // one hour ago
          start: (Math.floor(Date.now()/1000) - (60 * 60)).toString(),
        }
      })
      .then(response => {
        const newState = Object.assign({}, this.state, { data: response.data });
        this.setState(newState)

        setTimeout(() => {
          this.updateData()
        },5000);
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

    // update data
    this.updateData();
  }

  componentWillUnmount() {
    window.removeEventListener('resize', this.handleResize);
  }

  getTooltipLabel = datum => `${this.prettyDate(datum.AtTime)}\nQueries : ${this.getDataItem(datum, 'gudgeon-total-interval-queries')}\nBlocked: ${this.getDataItem(datum, 'gudgeon-blocked-interval-queries')}`;

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
      <div ref={this.containerRef}>
        <div className="chart-overflow">
          <ChartGroup containerComponent={container} height={100} width={width}>
            <ChartArea scale={{ x: "time", y: "linear" }} data={data} x={ (d) => d.AtTime } y={ (d) => this.getDataItem(d, 'gudgeon-total-interval-queries') } />
            <ChartArea scale={{ x: "time", y: "linear" }} data={data} x={ (d) => d.AtTime } y={ (d) => this.getDataItem(d, 'gudgeon-blocked-interval-queries') } />
          </ChartGroup>
        </div>
        <ChartLegend
          title="Actions Per Second"
          data={[{ name: 'Queries' }, { name: 'Blocked' }]}
          height={100}
          width={width}
        />
      </div>
    );
  }
}
