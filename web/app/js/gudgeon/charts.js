import React from 'react';
import Axios from 'axios';
import { 
  Card,
  CardItem,
  CardHeader,
  CardBody,
  Grid,
  GridItem
} from '@patternfly/react-core';
import { GudgeonChart } from './metrics-chart.js';
import { HumanBytes, LocaleNumber } from './helpers.js';
import gudgeonStyles from '../../css/gudgeon-app.css';
import { css } from '@patternfly/react-styles';

export class GudgeonCharts extends React.Component {
  constructor(props) {
    super(props);
  };

  ProcessorPercentFormatter = (value) => {
    return LocaleNumber(value / 1000) + "%"
  }

  queryMetrics = {
    "queries": {
      label: "Queries",
      formatter: LocaleNumber,
      series: {
        queries: { name: "Queries", key: "gudgeon-total-interval-queries" }, 
        blocked: { name: "Blocked", key: "gudgeon-blocked-interval-queries" } 
      }
    }
  }

  memoryMetrics = {
    "memory": {
      label: "Memory",
      formatter: HumanBytes,
      series: {
        heap: { name: "Allocated Heap", key: "gudgeon-allocated-bytes" }, 
        rss: { name: "Resident Memory", key: "gudgeon-process-used-bytes" } 
      }
    }
  }

  threadMetrics = {
    "threads": {
      label: "Threads",
      formatter: LocaleNumber,
      series: { 
        threads: { name: "Threads", key: "gudgeon-process-threads" },
        routines: { name: "Go Routines", key: "gudgeon-goroutines" } 
      }
    }
  }

  cpuMetrics = {
    "cpu": {
      label: "CPU",
      formatter: this.ProcessorPercentFormatter,
      domain: {
        maxY: 100000, // processor use is in 1000ths of a percent
        minY: 0
      },
      ticks: [50000, 100000],
      series: { 
        cpu: { name: "CPU Use", key: "gudgeon-cpu-hundreds-percent" } 
      }
    }    
  }

  state = {

  }  

  componentDidMount() {

  }  

  componentWillUnmount() {

  }
  
  render() {
    return (
      <React.Fragment>
        <Grid gutter="sm">
           <GridItem lg={6} md={6} sm={12}>
            <Card className={css(gudgeonStyles.maxHeight)}>
              <CardBody>
                <GudgeonChart metrics={ this.queryMetrics } chartName="query" />
              </CardBody>
            </Card>
          </GridItem>
           <GridItem lg={6} md={6} sm={12}>
            <Card className={css(gudgeonStyles.maxHeight)}>
              <CardBody>
                <GudgeonChart metrics={ this.memoryMetrics } chartName="memory" />
              </CardBody>
            </Card>
          </GridItem>
           <GridItem lg={6} md={6} sm={12}>
            <Card className={css(gudgeonStyles.maxHeight)}>
              <CardBody>
                <GudgeonChart metrics={ this.threadMetrics } chartName="thread" />
              </CardBody>
            </Card>
          </GridItem>
           <GridItem lg={6} md={6} sm={12}>
            <Card className={css(gudgeonStyles.maxHeight)}>
              <CardBody>
                <GudgeonChart metrics={ this.cpuMetrics } chartName="cpu" />
              </CardBody>
            </Card>
          </GridItem>                              
        </Grid>
      </React.Fragment>
    )
  }
}