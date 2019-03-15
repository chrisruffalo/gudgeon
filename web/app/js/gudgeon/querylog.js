import React from 'react';
import Axios from 'axios';
import {
  Button,
  GridItem,
  Card,
  CardBody,
} from '@patternfly/react-core';
import Icon from '@material-ui/core/Icon';
import { ErrorCircleOIcon } from '@patternfly/react-icons';
import MaterialTable from 'material-table';
import { PrettyDate } from './helpers.js';

export class QueryLog extends React.Component {
  constructor(props) {
    super(props);
  };

  updateData = () => {
    var { actions } = this.state
    if ( actions[0].iconProps.color == "primary" ) {
      actions[0].iconProps.color = "action"
    } else {
      actions[0].iconProps.color = "primary"
    }

    this.setState({ actions: actions })
  };

  dataQuery = query => new Promise((resolve, reject) => {
    // query variables
    var skip = query.page == 0 ? 0 : (query.page * query.pageSize);
    var after = (Math.floor(Date.now()/1000) - 60 * 60).toString()

    var params = {
      limit: query.pageSize,
      skip: skip,
      after: after
    }

    if ( query.search != null && query.search.length > 0 && query.search !== "" ) {
      params['responseText'] = query.search;
      params['clientName'] = query.search;
      params['rdomain'] = query.search;
      params['address'] = query.search;
    }

    if ( query.orderBy == null || query.orderBy.title == "Created" ) {
      params['sort'] = "created";
    } else {
      params['sort'] = "";
    }

    if ( query.orderDirection != null ) {
      params['direction'] = query.orderDirection;
    } else if ( params['sort'] == "created" ) {
      params['direction'] = "desc"; 
    } else {
      params['direction'] = "asc";
    }
    

    Axios
      .get('api/log',{ params: params })
      .then(response => response.data)
      .then(result => {
          resolve({
            data: result.items,
            page: query.page,
            totalCount: result.total
          });
      });
  });

  state = {
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
      icon: () => { return (<Icon>refresh</Icon>); },
      iconProps: {color: "primary"},
      isFreeAction: true,
      tooltip: "Refresh",
      onClick: this.updateData
    }],
    options: {
        pageSize: 10,
        pageSizeOptions: [ 5, 10, 20, 50, 100 ],
        showTitle: false,
        debounceInterval: 750
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
    const { columns, actions, options } = this.state;

    return (
      <React.Fragment>
        <GridItem lg={12} md={12} sm={12}>
          <MaterialTable columns={columns} data={this.dataQuery} actions={actions} options={options} />
        </GridItem>
      </React.Fragment>
    )
  }

}