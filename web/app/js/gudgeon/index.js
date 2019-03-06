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
  PageSectionVariants 
} from '@patternfly/react-core';
import { MetricsCards } from './metrics-cards.js';

export class Gudgeon extends React.Component {
  render() {
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

    // header glue
    const Header = (
      <PageHeader logo="Gudgeon"/>
    );

    return (
      <Page header={Header}>
        {NavigationBar}
        <PageSection>
          <MetricsCards />          
        </PageSection>
      </Page>
    );
  }
}