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
  SplitItem
} from '@patternfly/react-core';
import { MetricsCards } from './metrics-cards.js';
import gudgeonStyles from '../../css/gudgeon-app.css';
import { css } from '@patternfly/react-styles';

export class Gudgeon extends React.Component {
  state = {
    version: {
      'version': '',
      'longversion': '',
      'githash': ''
    }
  };

  componentWillMount() {
    var newVersion = window.version();
    const newState = Object.assign({}, this.state, { version: newVersion });
    this.setState(newState)
  }

  render() {
    var { version } = this.state

    // header navigation
    const NavigationBar = (
      <div style={{ backgroundColor: '#292e34', paddingLeft: '1rem', paddingRight: '1rem' }}>
        <Nav>
          <NavList variant={NavVariants.horizontal}>
            <NavItem preventDefault to="#dashboard" itemId={0}>
              Dashboard
            </NavItem>
            <NavItem preventDefault to="#qlog" itemId={1}>
              Query Log
            </NavItem>
          </NavList>
        </Nav>
      </div>      
    );

    const Footer = (
      <div style={{ backgroundColor: '#292e34', padding: '1rem', color: '#ffffff' }}>
      <Split gutter="sm">
        <SplitItem isMain>
          <p class={css(gudgeonStyles.footerText)}>&copy; Chris Ruffalo 2019</p>
          <p class={css(gudgeonStyles.footerText)}><a href="https://github.com/chrisruffalo/gudgeon">@GitHub</a></p>
        </SplitItem>
        <SplitItem><p class={css(gudgeonStyles.footerText)}>{ version.longversion }</p></SplitItem>
      </Split>      
      </div>      
    );    

    // header glue
    const Header = (
      <PageHeader logo="Gudgeon" />
    );

    return (
      <div className={css(gudgeonStyles.maxHeight)}>
        <Page header={Header} className={css(gudgeonStyles.maxHeight)}>
          {NavigationBar}
          <PageSection>
            <MetricsCards />          
          </PageSection>
          { Footer }
        </Page>
      </div>
    );
  }
}