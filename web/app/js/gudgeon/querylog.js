import React from 'react';
import Axios from 'axios';
import {
  GridItem,
  Card,
  CardBody,
} from '@patternfly/react-core';
import MaterialTable from 'material-table';
import { PrettyDate } from './helpers.js';

export class QueryLog extends React.Component {
  constructor(props) {
    super(props);
  };

  updateData = () => {
    var { actions } = this.state
    actions[0].iconProps.color = "action"
    this.setState({ actions: actions })
  };

  dataQuery = (query) => {
    new Promise((resolve, reject) => {
      // query variables
      var limit = 1000;
      var skip = query.page == 0 ? 0 : ((query.page - 1) * query.pageSize);
      var after = (Math.floor(Date.now()/1000) - timeRangeValue).toString()

      // build url
      var url = 'api/log?limit=' + limit + "&skip=" + skip + "&after=" + after;

      // make query
      fetch(url)
        .then(response => response.json())
        .then(result => {
          resolve({
            data: result.data.items,
            page: result.page - 1,
            totalCount: result.data.total
          });
        });
    });
  };

  state = {
    refreshing: false,
    columns: [
      { title: 'Client', 
        searchable: true,
        sorting: false,        
        render: rowData => {
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
      { title: 'Request',
        searchable: true,
        sorting: false,        
        render: rowData => {
          return (
            <div>{ rowData.RequestDomain } ({ rowData.RequestType })</div>
          );
        }
      },
      { title: 'Response', 
        searchable: true,
        sorting: false,
        render: rowData => {
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
      { title: 'Created',
        searchable: false,
        sorting: true,
        defaultSort: "desc",
        render: rowData => {
          return (
            <div>{ PrettyDate(rowData.Created) }</div>
          );
        }
      }
    ],
    data: [],
    actions: [{
      disabled: false,
      icon: "refresh",
      iconProps: {color: "primary"},
      isFreeAction: true,
      tooltip: "Refresh",
      onClick: this.updateData
    }],
    options: {
        pageSize: 10,
        pageSizeOptions: [ 5, 10, 20, 50, 100 ],
        showTitle: false,
        debounceInterval: 350
    }
  };

  componentDidMount() {

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
    const { columns, dataQuery, actions, options } = this.state;

    return (
      <React.Fragment>
        <GridItem lg={12} md={12} sm={12}>
          <MaterialTable columns={columns} data={dataQuery} actions={actions} options={options} />
        </GridItem>
      </React.Fragment>
    )
  }

}