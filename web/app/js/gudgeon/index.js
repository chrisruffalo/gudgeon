import React from 'react';
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
  state = {
    version: {
      'version': '',
      'longversion': '',
      'githash': ''
    },
    activeItem: -1
  };

  onSelect = result => {
    this.setState({
      activeItem: result.itemId
    });
  };

  componentWillMount() {
    var newVersion = window.version();
    const newState = Object.assign({}, this.state, { version: newVersion });
    this.setState(newState)
  };

  onSelect = result => {
    this.setState({
      activeItem: result.itemId
    });
  };

  render() {
    const { version } = this.state
    var { activeItem } = this.state;

    var NavItems = [];
    if ( window.config().metrics ) {
      if ( activeItem < 0 ) {
        activeItem = 0
      }
      NavItems.push(<NavItem preventDefault to="#dashboard" key="dashboard" itemId={0} isActive={activeItem === 0}>Dashboard</NavItem>);
    }
    if ( window.config().query_log ) {
      if ( activeItem < 0 ) {
        activeItem = 1
      }      
      NavItems.push(<NavItem preventDefault to="#qlog" key="qlog" itemId={1} isActive={activeItem === 1}>Query Log</NavItem>);
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
            <p className={css(gudgeonStyles.footerText)}>{ version.version }</p>
            <p className={css(gudgeonStyles.footerText)}>git@{ version.githash }</p>
          </SplitItem>
        </Split>      
      </div>      
    );

    return (
      <div className={css(gudgeonStyles.maxHeight)}>
        <Page header={Header} className={css(gudgeonStyles.maxHeight)}>
          <PageSection>
            { NavItems.length === 0 ? EmptyState : null }
            { window.config().metrics && activeItem === 0 ? <MetricsCards /> : null }
            { window.config().query_log && activeItem === 1 ? <QueryLog /> : null }
          </PageSection>
          { Footer }
        </Page>
      </div>
    );
  }
}