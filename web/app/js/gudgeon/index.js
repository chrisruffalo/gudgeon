import React from 'react';
import { 
  Nav, 
  NavItem,
  NavList,
  NavVariants,
  Page, 
  PageHeader, 
  PageSection, 
  PageSectionVariants 
} from '@patternfly/react-core';
import { QPSChart } from './metrics-chart.js'

export class HorizontalPage extends React.Component {
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
          <QPSChart />
        </PageSection>
      </Page>
    );
  }
}