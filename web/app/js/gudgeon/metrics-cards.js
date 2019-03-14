import React from 'react';
import Axios from 'axios';
import { 
  Card,
  CardItem,
  CardHeader,
  CardBody,
  Grid,
  GridItem,
  DataList,
  DataListItem,
  DataListCell,
  FormSelect,
  FormSelectOption
} from '@patternfly/react-core';
import { 
  Table, 
  TableHeader, 
  TableBody, 
  TableVariant 
} from '@patternfly/react-table';
import { QPSChart } from './metrics-chart.js';
import { HumanBytes, LocaleNumber } from './helpers.js';
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
    currentMetrics: 'lifetime',
    rows: [],
    timer: null
  };  

  options = [
    { value: 'lifetime', label: 'Lifetime', disabled: false},
    { value: 'session', label: 'Session', disabled: false }
  ];

  onQueryMetricsOptionChange = (value, event) => {
    this.setState({ currentMetrics: value });
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
      this.setState({ timer: null })
    }
  }
  
  render() {
    const { columns, data, rows, currentMetrics } = this.state;

    return (
      <Grid gutter="sm">
        <GridItem lg={4} md={6} sm={12}>
          <Card className={css(gudgeonStyles.maxHeight)}>
            <CardBody>
              <div style={{ "paddingBottom": "15px" }}>
                <FormSelect value={this.state.currentMetrics} onChange={this.onQueryMetricsOptionChange} aria-label="FormSelect Input">
                  {this.options.map((option, index) => (
                    <FormSelectOption isDisabled={option.disabled} key={index} value={option.value} label={option.label} />
                  ))}
                </FormSelect>
              </div>
              <DataList aria-label="Lifetime Metrics">
                <DataListItem aria-labelledby="simple-item1">
                  <DataListCell>Queries</DataListCell>
                  <DataListCell>{ LocaleNumber(this.getDataMetric(data, 'total-' + currentMetrics + '-queries')) } </DataListCell>
                </DataListItem>
                <DataListItem aria-labelledby="simple-item2">
                  <DataListCell>Blocked</DataListCell>
                  <DataListCell>{ LocaleNumber(this.getDataMetric(data, 'blocked-' + currentMetrics + '-queries')) }</DataListCell>
                </DataListItem>
              </DataList>            
            </CardBody>
          </Card>          
        </GridItem>
        <GridItem lg={4} md={6} sm={12}>
          <Card className={css(gudgeonStyles.maxHeight)}>
            <CardBody>
              <QPSChart metrics={[ { name: "Queries", key: "gudgeon-total-interval-queries" }, { name: "Blocked", key: "gudgeon-blocked-interval-queries" } ]} formatter = { LocaleNumber }/>
            </CardBody>
          </Card>
        </GridItem>
        <GridItem lg={4} md={6} sm={12}>
          <Card className={css(gudgeonStyles.maxHeight)}>
            <CardBody>
              <QPSChart metrics={[ { name: "Allocated Heap", key: "gudgeon-allocated-bytes" }, { name: "Resident Memory", key: "gudgeon-process-used-bytes" } ]} formatter = { HumanBytes } />
            </CardBody>
          </Card>
        </GridItem>
        <GridItem lg={6} md={6} sm={12}>
          <Card className={css(gudgeonStyles.maxHeight)}>
            <CardBody>
              <Table aria-label="Block Lists" variant={TableVariant.compact} cells={columns} rows={rows}>
                <TableHeader />
                <TableBody />
              </Table>            
            </CardBody>
          </Card>          
        </GridItem>
      </Grid>
    )
  }
}