import React from 'react';
import { BrowserRouter as Router, Route, Switch, Redirect, Link } from "react-router-dom";
import { 
  Card,
  CardItem,
  CardHeader,
  CardBody,
  Grid,
  GridItem,
  Nav, 
  NavItem,
  NavList,
  NavVariants,
  Page, 
  PageHeader, 
  PageSection, 
  PageSectionVariants,
  Split,
  SplitItem,
} from '@patternfly/react-core';
import { CubesIcon } from '@patternfly/react-icons';
import { MetricsCards } from './metrics-cards.js';
import { QueryLog } from './querylog.js';
import { QueryTest } from './querytest.js';
import gudgeonStyles from '../../css/gudgeon-app.css';
import { css } from '@patternfly/react-styles';

export class Gudgeon extends React.Component {
  // empty state
  state = {};

  componentWillMount() {

  };

  render() {
    const { version } = this.state
    var defaultRoute = "";

    var NavItems = [];
    if ( window.config().metrics ) {
      if ( defaultRoute == "" ) {
        defaultRoute = "/metrics"
      }
      NavItems.push(<NavItem to="#metrics" key="metrics"><Link to="/metrics">Metrics</Link></NavItem>);
    }
    if ( window.config().query_log ) {
      if ( defaultRoute == "" ) {
        defaultRoute = "/qlog"
      }
      NavItems.push(<NavItem to="#qlog" key="qlog"><Link to="/qlog">Query Log</Link></NavItem>);
    }

    // header navigation
    const NavigationBar = (
      <div style={{ backgroundColor: '#292e34', padding: '1rem' }}>
        <Nav onSelect={this.onSelect}>
          <NavList variant={NavVariants.horizontal}>
            {NavItems.length > 1 ? NavItems : null}
          </NavList>
        </Nav>
      </div>      
    );

    // header glue
    const Header = (
      <PageHeader style={{ backgroundColor: '#292e34', color: '#ffffff' }} topNav={NavigationBar} logo="Gudgeon" />
    );

    const EmptyState = (null);

    const Footer = (
      <div style={{ backgroundColor: '#292e34', padding: '1rem', color: '#ffffff' }}>
        <Split gutter="sm">
          <SplitItem isMain>
            <p className={css(gudgeonStyles.footerText)}>&copy; Chris Ruffalo 2019</p>
            <p className={css(gudgeonStyles.footerText)}><a href="https://github.com/chrisruffalo/gudgeon">@GitHub</a></p>
          </SplitItem>
          <SplitItem>
            <p className={css(gudgeonStyles.footerText)}>{ window.version().version }</p>
            <p className={css(gudgeonStyles.footerText)}>git@{ window.version().githash }</p>
          </SplitItem>
        </Split>      
      </div>      
    );

    const Metrics = window.config().metrics ? ( <MetricsCards /> ) : null;
    const QLog = window.config().query_log ? ( <QueryLog />) : null;

    return (
      <div className={css(gudgeonStyles.maxHeight)}>
        <Router>
          <Page header={Header} className={css(gudgeonStyles.maxHeight)}>
            <PageSection>
              <Switch>              
                { window.config().metrics ? <Route path="/metrics" component={ () => Metrics } /> : null }
                { window.config().query_log ? <Route path="/qlog" component={ () => QLog } /> : null }
                <Redirect from="/" to={ defaultRoute } />
              </Switch>
            </PageSection>
            { Footer }
          </Page>
        </Router>
      </div>
    );
  }
}