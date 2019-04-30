import React from 'react';
import Axios from 'axios';
import {
  GridItem,
} from '@patternfly/react-core';
import { ErrorCircleOIcon, VolumeIcon } from '@patternfly/react-icons';
import MaterialTable from 'material-table';
import { PrettyDate } from './helpers.js';

export class QueryLog extends React.Component {
  constructor(props) {
    super(props);
    this.state.externalSearch = this.props.externalSearch || false;
    if ( this.props.externalSearch ) {
      this.state.externalKey = this.props.externalKey;
      this.state.externalQuery = this.props.externalQuery;
    }
  };

  state = {
    columns: [
      { title: 'Client', 
        searchable: true,
        sorting: false,        
        render: rowData => {
          let clientString = "";
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
          let responseText = rowData.ResponseText;
          let responseCode = rowData.Rcode;
          if ( responseText == null || responseText === "" || responseText.length < 0) {
            if (responseCode != null && "" !== responseCode && "NOERROR" !== responseCode ) {
              responseText = responseCode
            } else {
              responseText = "( EMPTY )"
            }
          }

          if ( rowData.Blocked ) {
            return (
              <div style={{ color: "red" }}><ErrorCircleOIcon alt="blocked" /> BLOCKED</div>
            );
          } else if ( rowData.Match === 1 ) {
            return (
              <div style={{ color: "red" }}><ErrorCircleOIcon alt="blocked" /> { rowData.MatchList }{ rowData.MatchRule ? ' (' + rowData.MatchRule + ")" : null }</div>
            );          
          } else if ( rowData.Cached ) {
            return (
              <div style={{ color: "green" }}><VolumeIcon alt="cached" /> { responseText }</div>
            );
          } else {
            return (
              <div>{ responseText }</div>
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
    actions: [],
    options: {
        pageSize: 10,
        pageSizeOptions: [ 5, 10, 20, 50, 100 ],
        showTitle: false,
        debounceInterval: 750,
        search: false,
        toolbar: false
    }
  };

  dataQuery = query => new Promise((resolve, reject) => {
    // query variables
    let skip = query.page === 0 ? 0 : (query.page * query.pageSize);
    let after = (Math.floor(Date.now()/1000) - 60 * 60).toString();

    let params = {
      limit: query.pageSize,
      skip: skip,
      after: after
    };

    // load search from external source, will be overwritten if search comes behind
    let { externalSearch, externalKey, externalQuery } = this.state;
    if ( externalSearch ) {
      // set param
      params[externalKey] = externalQuery;
      // all time
      after = null;
    }

    if ( query.orderBy == null || query.orderBy.title === "Created" ) {
      params['sort'] = "created";
      if ( query.orderBy == null ) {
        params['direction'] = "desc";
      } else {
        params['direction'] = query.orderDirection == null ? "asc" : query.orderDirection ; 
      }
    } else {
      params['sort'] = query.orderBy.toLowerCase();
    }

    if ( params['direction'] == null && query.orderDirection != null && query.orderDirection !== "" ) {
      params['direction'] = query.orderDirection;
    }

    Axios
      .get('/api/query/list',{ params: params })
      .then(response => response.data)
      .then(result => {
          resolve({
            data: result.items,
            page: query.page,
            totalCount: result.total
          });
      });
  });

  componentDidMount() {

  }

  componentWillUnmount() {
    // stop updating if timer is not null
    let { autoRefreshTimer } = this.state;
    if ( autoRefreshTimer != null ) {
      clearTimeout(autoRefreshTimer);
      this.setState({ autoRefreshTimer: null })             
    }
  }

  render() {
    const { columns, actions, options } = this.state;

    return (
      <GridItem lg={12} md={12} sm={12}>
        <MaterialTable title={null} columns={columns} data={this.dataQuery} actions={actions} options={options} />
      </GridItem>
    )
  }

}