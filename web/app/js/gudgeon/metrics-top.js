import React from 'react';
import Axios from 'axios';
import { 
  DataList,
  DataListItem,
  DataListCell
} from '@patternfly/react-core';
import { LocaleNumber } from './helpers.js';
import gudgeonStyles from '../../css/gudgeon-app.css';
import { css } from '@patternfly/react-styles';

export class MetricsTopList extends React.Component {
  constructor(props) {
    super(props);
  };

  state = {
    data: [],
    timer: null
  };  

  updateData() {
    var { timer } = this.state
    if ( timer != null ) {
      clearTimeout(timer)
    }

    Axios
      .get("/api/metrics/top/" + this.props.topType )
      .then(response => {
        this.setState({ data: response.data })
        var newTimer = setTimeout(() => {
          this.updateData()
        },15000); // update every 15s

        // update the data in the state
        this.setState({ timer: newTimer })
      }).catch((error) => {
        var newTimer = setTimeout(() => {
          this.updateData()
        },60000); // on error try again in 60s

        // update the data in the state
        this.setState({ timer: newTimer })
      });
  }

  componentDidMount() {
    this.updateData()
  }  

  componentWillUnmount() {
    var { timer } = this.state
    if ( timer != null ) {
      clearTimeout(timer)
    }
  }
  
  render() {
    // build data list items from data
    var { data } = this.state
    const ListItems = data.map((item, index) => {
      return (
          <DataListItem key={ item.Desc } aria-labelledby={ "label-" + index }>
            <DataListCell className={css(gudgeonStyles.smallCell)} width={2}><span className={css(gudgeonStyles.leftCard)} id={ "label-" + index }>{ item.Desc }</span></DataListCell>
            <DataListCell className={css(gudgeonStyles.smallCell)} width={1}><div className={css(gudgeonStyles.rightCard)} >{ LocaleNumber(item.Count) }</div></DataListCell>
          </DataListItem>        
      );
    });

    return (
      <React.Fragment>
        <DataList aria-label="Lifetime Metrics">
          {ListItems}
        </DataList>
      </React.Fragment>
    )
  }
}