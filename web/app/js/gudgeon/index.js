import React from 'react';
import { BrowserRouter as Router, Route, Switch, Redirect, NavLink as Link } from "react-router-dom";
import {
  Brand,
  Nav,
  NavItem,
  NavList,
  NavVariants,
  Page,
  PageHeader,
  PageSection,
  Split,
  SplitItem,
  EmptyState,
  EmptyStateIcon,
  EmptyStateBody,
  Title, Grid, GridItem, Card, CardHeader, CardBody
} from '@patternfly/react-core';
import { CubesIcon } from '@patternfly/react-icons';
import { Dashboard } from './dashboard.js';
import { GudgeonCharts } from './charts.js';
import { QueryLog } from './qlog-table.js';
import { QueryTester } from './query-tester.js';
import gudgeonStyles from '../../css/gudgeon-app.css';
import { css } from '@patternfly/react-styles';
import {MetricsTopList} from "./metrics-top";

export class Gudgeon extends React.Component {
  // empty state
  state = {};

  componentWillMount() {

  }

  componentWillUnmount() {

  }

  createDetailedQlog = ({ location, match }) => {
    if ( !window.config().query_log || !window.config().query_log_persist ) {
      return null;
    }

    // get search params
    let sparams = new URLSearchParams(location.search);

    // handle external search parameters
    if ( sparams.get("st") && sparams.get("query") ) {
      return <QueryLog externalSearch={true} externalKey={sparams.get("st")} externalQuery={sparams.get("query")} />;
    }
    // otherwise return vanilla query log
    return <QueryLog />;
  };

  expandedMetricsView = ({ match }) => {
    const titleMap = {
      "clients": "Top Clients",
      "rules": "Top Rule Matches",
      "domains": "Top Queried Domains"
    };
    return (
        <Grid gutter="sm">
          <GridItem lg={12} md={12} sm={12}>
            <Card className={css(gudgeonStyles.maxHeight)}>
              <CardHeader>
                <Split gutter="sm">
                  <SplitItem isFilled={true} style={{ width: "100%" }}>
                    { titleMap[match.params.topType] }
                  </SplitItem>
                  <SplitItem isFilled={true} style={{ textAlign: "right" }}>
                    <Link to={ "/dashboard" } >&lt;dashboard</Link>
                  </SplitItem>
                </Split>
              </CardHeader>
              <CardBody>
                <MetricsTopList topType={ match.params.topType } limit={50} />
              </CardBody>
            </Card>
          </GridItem>
        </Grid>
    );
  };

  render() {
    var defaultRoute = "";
    var NavItems = [];
    if ( window.config().metrics ) {
      NavItems.push(<NavItem to="#dashboard" key="dashboard"><Link activeClassName="pf-m-current" to="/dashboard">Dashboard</Link></NavItem>);
      if ( window.config().metrics_persist ) {
        NavItems.push(<NavItem to="#charts" key="charts"><Link activeClassName="pf-m-current" to="/charts">Charts</Link></NavItem>);
      }
      if ( defaultRoute === "" ) {
        defaultRoute = "/dashboard"
      }
    }
    if ( window.config().query_log && window.config().query_log_persist ) {
      NavItems.push(<NavItem to="#qlog" key="qlog"><Link activeClassName="pf-m-current" to="/qlog">Query Log</Link></NavItem>);
      if ( defaultRoute === "" ) {
        defaultRoute = "/qlog"
      }
    }
    NavItems.push(<NavItem to="#qtest" key="qtest"><Link activeClassName="pf-m-current" to="/query_test">Query Test</Link></NavItem>);
    if ( defaultRoute === "" ) {
      defaultRoute = "/query_test"
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
      <PageHeader style={{ backgroundColor: '#292e34', color: '#ffffff' }} topNav={NavigationBar} logo={ <Brand src="../../img/gudgeon_logo.svg" alt="Gudgeon" /> } />
    );

    const NoFeaturesEnabled = (
      <center>
        <EmptyState>
          <EmptyStateIcon icon={ CubesIcon } />
          <Title headingLevel="h5" size="lg">No Features Enabled</Title>
          <EmptyStateBody>
            No features have been enabled in Gudgeon. See your configuration yaml and enable the Metrics or Query Log features.
          </EmptyStateBody>
        </EmptyState>      
      </center>
    );

    const Footer = (
      <div style={{ backgroundColor: '#292e34', padding: '1rem', color: '#ffffff' }}>
        <Split gutter="sm">
          <SplitItem isFilled={true} style={{ width: "100%" }}>
            <p className={css(gudgeonStyles.footerText)}>&copy; Chris Ruffalo 2019</p>
            <p className={css(gudgeonStyles.footerText)}><a href="https://github.com/chrisruffalo/gudgeon">@GitHub</a></p>
          </SplitItem>
          <SplitItem isFilled={true} style={{ textAlign: "right" }}>
            <p className={css(gudgeonStyles.footerText)}>{ window.version().version }{ window.version().release !== "" ? "-" + window.version().release : ""}</p>
            <p className={css(gudgeonStyles.footerText)}><a href={ "https://github.com/chrisruffalo/gudgeon/tree/" + window.version().githash }>git@{ window.version().githash }</a></p>
          </SplitItem>
        </Split>      
      </div>      
    );

    // if the default route is blank we don't want to show anything but this
    const Catcher = defaultRoute === "" ? ( <Route component={ () => NoFeaturesEnabled } /> ) : ( <Redirect to={ defaultRoute } /> );

    const Dboard = window.config().metrics ? ( <Dashboard /> ) : null;
    const Charts = window.config().metrics ? ( <GudgeonCharts /> ) : null;
    const QTest = ( <QueryTester /> );

    return (
      <div className={css(gudgeonStyles.maxHeight)}>
        <Router>
          <Page header={Header} className={css(gudgeonStyles.maxHeight)}>
            <PageSection>
              <Switch>
                { window.config().metrics ? <Route path="/dashboard" component={ () => Dboard } /> : null }
                { window.config().metrics && window.config().metrics_persist ? <Route path="/charts" component={ () => Charts } /> : null }
                { window.config().metrics && window.config().metrics_persist ? <Route path="/top/:topType" component={ this.expandedMetricsView } /> : null }
                { window.config().query_log && window.config().query_log_persist ? <Route path="/qlog" component={ this.createDetailedQlog } /> : null }
                <Route path="/query_test" component={ () => QTest } />
                { Catcher }
              </Switch>
            </PageSection>
            { Footer }
          </Page>
        </Router>
      </div>
    );
  }
}