import React from 'react';
import Axios from 'axios';
import { 
  Card,
  CardItem,
  CardHeader,
  CardBody,
  DataList,
  DataListItem,
  DataListCell,
  Grid,
  GridItem,
  Split,
  SplitItem
} from '@patternfly/react-core';
import { 
  Table, 
  TableHeader, 
  TableBody, 
  TableVariant 
} from '@patternfly/react-table';
import { QPSChart } from './metrics-chart.js'

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
    rows: []
  };  

  updateData() {
    Axios
      .get("api/metrics/current")
      .then(response => {
        this.updateMetricsState(response.data)

        setTimeout(() => {
          this.updateData()
        },1000); // update every second
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
      if ( data.metrics['gudgeon-rules-list-' + key ] != null ){
        newRow.push(data.metrics['gudgeon-rules-list-' + key ].count);
      } else {
        return;
      }

      if ( data.metrics['gudgeon-rules-blocked-' + key] != null ) {
        newRow.push(data.metrics['gudgeon-rules-blocked-' + key].count);
      } else {
        newRow.push(0);
      }
      rows.push(newRow);
    });

    // update the data in the state
    const newState = Object.assign({}, this.state, { data: data, rows: rows });
    this.setState(newState)
  }

  componentDidMount() {
    // update data
    this.updateData();
  }  
  
  render() {
    const { columns, rows } = this.state;

    return (
      <Grid gutter="md">
        <GridItem span={4} lg={3} md={6} sm={12}>
          <Card>
            <CardHeader>Query Metrics</CardHeader>
            <CardBody>
              Lifetime Queries {this.state.data.metrics['gudgeon-total-lifetime-queries'].count } <br/>
              Lifetime Blocks {this.state.data.metrics['gudgeon-blocked-lifetime-queries'].count } <br/>
              Session Queries {this.state.data.metrics['gudgeon-total-session-queries'].count } <br/>
              Session Blocks {this.state.data.metrics['gudgeon-blocked-session-queries'].count }
            </CardBody>
          </Card>          
        </GridItem>
        <GridItem span={4} lg={3} md={6} sm={12}>
          <Card>
            <CardBody>
              <Table aria-label="Block Lists" variant={TableVariant.compact} cells={columns} rows={rows}>
                <TableHeader />
                <TableBody />
              </Table>            
            </CardBody>
          </Card>          
        </GridItem>
        <GridItem span={4} lg={6} md={12} sm={12}>
          <Card>
            <CardBody>
              <QPSChart />
            </CardBody>
          </Card>
        </GridItem>
      </Grid>
    )
  }
}