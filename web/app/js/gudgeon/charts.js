import React from 'react';
import {
  Card,
  CardBody,
  Grid,
  GridItem
} from '@patternfly/react-core';
import { Metrics, GudgeonChart } from './metrics-chart.js';

export class GudgeonCharts extends React.Component {
  constructor(props) {
    super(props);
  };

  state = {};

  componentDidMount() {

  }  

  componentWillUnmount() {

  }
  
  render() {
    return (
      <React.Fragment>
        <Grid gutter="sm">
           <GridItem lg={6} md={6} sm={12}>
            <Card className={"maxHeight"}>
              <CardBody>
                <GudgeonChart metrics={ [Metrics.Queries] } chartName="query" />
              </CardBody>
            </Card>
          </GridItem>
           <GridItem lg={6} md={6} sm={12}>
            <Card className={"maxHeight"}>
              <CardBody>
                <GudgeonChart metrics={ [Metrics.Memory] } chartName="memory" />
              </CardBody>
            </Card>
          </GridItem>
           <GridItem lg={6} md={6} sm={12}>
            <Card className={"maxHeight"}>
              <CardBody>
                <GudgeonChart metrics={ [Metrics.Threads] } chartName="thread" />
              </CardBody>
            </Card>
          </GridItem>
           <GridItem lg={6} md={6} sm={12}>
            <Card className={"maxHeight"}>
              <CardBody>
                <GudgeonChart metrics={ [Metrics.CPU] } chartName="cpu" />
              </CardBody>
            </Card>
          </GridItem>                              
        </Grid>
      </React.Fragment>
    )
  }
}