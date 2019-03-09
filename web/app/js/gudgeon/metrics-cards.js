import React from 'react';
import Axios from 'axios';
import { 
  Card,
  CardItem,
  CardHeader,
  CardBody,
  Grid,
  GridItem,
} from '@patternfly/react-core';
import { 
  Table, 
  TableHeader, 
  TableBody, 
  TableVariant 
} from '@patternfly/react-table';
import { QPSChart } from './metrics-chart.js';
import gudgeonStyles from '../../css/gudgeon-app.css';
import { css } from '@patternfly/react-styles';

export class MetricsCards extends React.Component {
  constructor(props) {
    super(props);
  };

  state = {
    width: 0,
    data: {
      'metrics': {
        'gudgeon-blocked-lifetime-queries': {
          'count': 0
        },
        'gudgeon-blocked-session-queries': {
          'count': 0
        },
        'gudgeon-total-lifetime-queries': {
          'count': 0
        },
        'gudgeon-total-session-queries': {
          'count': 0
        }
      },
      'lists': []
    },
    columns: [
      'List',
      'Rules',
      'Blocked'
    ],
    rows: [],
    timer: null
  };  

  updateData() {
    Axios
      .get("api/metrics/current")
      .then(response => {
        this.setState({ data: response.data })
        this.updateMetricsState(response.data)

        var newTimer = setTimeout(() => {
          this.updateData()
        },2000); // update every 2s

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

  updateMetricsState(data) {
    // update the rows by building each
    var rows = [];
    data.lists.forEach((element) => {
      if ( element['name'] == null ) {
        return;
      }
      var newRow = [];
      newRow.push(element['name'])
      var key = element['short']
      newRow.push(this.getDataMetric(data, 'rules-list-' + key));
      newRow.push(this.getDataMetric(data, 'rules-blocked-' + key));
      rows.push(newRow);
    });

    // update the data in the state
    this.setState({ rows: rows })
  }

  getDataMetric(data, key) {
    if ( data.metrics == null ) {
      return 0
    }
    if ( data.metrics["gudgeon-" + key] == null ) {
      return 0
    }
    return data.metrics["gudgeon-" + key].count
  }

  componentDidMount() {
    // update data
    this.updateData();
  }  

  componentWillUnmount() {
    var { timer } = this.state
    if ( timer != null ) {
      clearTimeout(timer)
      const newState = Object.assign({}, this.state, { timer: null });
      this.setState(newState)
    }
  }
  
  render() {
    const { columns, data, rows } = this.state;

    return (
      <Grid gutter="md">
        <GridItem lg={3} md={6} sm={12}>
          <Card className={css(gudgeonStyles.maxHeight)}>
            <CardHeader>Query Metrics</CardHeader>
            <CardBody>
              <p>Lifetime Queries { this.getDataMetric(data, 'total-lifetime-queries') }</p>
              <p>Lifetime Blocks { this.getDataMetric(data, 'blocked-lifetime-queries') }</p>
              <p>Session Queries { this.getDataMetric(data, 'total-session-queries') }</p>
              <p>Session Blocks { this.getDataMetric(data, 'blocked-session-queries') }</p>
            </CardBody>
          </Card>          
        </GridItem>
        <GridItem lg={3} md={6} sm={12}>
          <Card className={css(gudgeonStyles.maxHeight)}>
            <CardBody>
              <Table aria-label="Block Lists" variant={TableVariant.compact} cells={columns} rows={rows}>
                <TableHeader />
                <TableBody />
              </Table>            
            </CardBody>
          </Card>          
        </GridItem>
        <GridItem lg={6} md={12} sm={12}>
          <Card className={css(gudgeonStyles.maxHeight)}>
            <CardBody>
              <QPSChart />
            </CardBody>
          </Card>
        </GridItem>
      </Grid>
    )
  }
}