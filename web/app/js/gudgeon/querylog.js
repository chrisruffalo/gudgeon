import React from 'react';
import Axios from 'axios';
import {
  GridItem,
  Card,
  CardBody,
} from '@patternfly/react-core';
import MaterialTable from 'material-table'
import { PrettyDate } from './helpers.js';
import { QPSChart } from './metrics-chart.js';
import gudgeonStyles from '../../css/gudgeon-app.css';
import { css } from '@patternfly/react-styles';
import { RebootingIcon, ErrorCircleOIcon } from '@patternfly/react-icons';

export class QueryLog extends React.Component {
  constructor(props) {
    super(props);
  };

  timeRangeOptions = [
    { value: (60 * 60), label: 'Last Hour', disabled: false },
    { value: (60 * 60) * 6, label: 'Last 6 Hours', disabled: false },
    { value: (60 * 60) * 12, label: 'Last 12 Hours', disabled: false },
    { value: (60 * 60) * 24, label: 'Last 24 Hours', disabled: false },
    { value: -1, label: 'All Time', disabled: false }
  ];

  viewItemOptions = [
    { value: 10, label: '10', disabled: false },
    { value: 25, label: '25', disabled: false },
    { value: 50, label: '50', disabled: false },
    { value: 100, label: '100', disabled: false },
    { value: 500, label: '500', disabled: false },
    { value: 'none', label: 'All', disabled: false }
  ];

  state = {
    Queries: [],
    columns: [
      { title: 'Client', render: rowData => {
          var clientString = "";
          if ( rowData.ClientName ) {
            clientString = clientString + rowData.ClientName + " | ";
          }
          clientString = clientString + rowData.Address;
          return (
            <div>{ clientString }</div>
          );
        }
      },
      { title: 'Request', render: rowData => {
          return (
            <div>{ rowData.RequestDomain } ({ rowData.RequestType })</div>
          );
        }
      },
      { title: 'Response', render: rowData => {
          if ( rowData.Blocked ) {
            return (
              <div><ErrorCircleOIcon /> { rowData.BlockedList }{ rowData.BlockedRule ? ' (' + rowData.BlockedRule + ")" : null }</div>
            );
          } else {
            return (
              <div>{ rowData.ResponseText }</div>
            );
          }
        }
      },
      { title: 'Created', render: rowData => {
          return (
            <div>{ PrettyDate(rowData.Created) }</div>
          );
        }
      }
    ],
    data: [],
    limit: 10,
    page: 0,
    isOpen: false,
    isQuerying: false,
    autoRefresh: false,
    autoRefreshTimer: null,
    timeRangeValue: this.timeRangeOptions[0].value,
    pageCountValue: this.viewItemOptions[0].value
  };

  updateData(triggerAuto) {
    var { limit, page, autoRefreshTimer, autoRefresh, timeRangeValue, pageCountValue } = this.state

    // clear the timeout when entering this method
    if ( autoRefreshTimer != null ) {
      clearTimeout(autoRefreshTimer)
    }

    if ( triggerAuto == null ) {
      triggerAuto = autoRefresh
    }

    // update the data in the state
    this.setState({ isQuerying: true })       

    // set time if set to a value
    var queryParams = {}
    if ( timeRangeValue >= 0 ) {
      queryParams['after'] = (Math.floor(Date.now()/1000) - timeRangeValue).toString()
    }
    
    // set limit
    queryParams['limit'] = pageCountValue

    Axios
      .get("api/log", {
        params: queryParams
      })
      .then(response => {
        var newTimer = null;
        if ( triggerAuto ) {
          // set timeout and update data again
          newTimer = setTimeout(() => {
            this.updateData()
          },2000); // two seconds
        }

        // update the data in the state
        this.setState({ autoRefreshTimer: newTimer, isQuerying: false, data: response.data})       
      });
  }

  componentDidMount() {
    // update data
    this.updateData();
  }

  componentWillUnmount() {
    // stop updating if timer is not null
    var { autoRefreshTimer } = this.state
    if ( autoRefreshTimer != null ) {
      clearTimeout(autoRefreshTimer)
      this.setState({ autoRefreshTimer: null })             
    }
  }

  render() {
    const { columns, data } = this.state;

    return (
      <React.Fragment>
        <GridItem lg={12} md={12} sm={12}>
          <MaterialTable columns={columns} data={data} />
        </GridItem>
      </React.Fragment>
    )
  }

}