// import style
import '@patternfly/react-core/dist/styles/base.css';

// additional patternfly styles
import accessibleStyles from '@patternfly/patternfly/utilities/Accessibility/accessibility.css';
import spacingStyles from '@patternfly/patternfly/utilities/Spacing/spacing.css';

// import react base
import React from 'react';
import ReactDOM from 'react-dom';

// import navigation
import { Nav, NavList, NavItem, NavVariants } from '@patternfly/react-core';

class NavHorizontalList extends React.Component {
  state = {
    activeItem: 0
  };

  onSelect = result => {
    this.setState({
      activeItem: result.itemId
    });
  };

  render() {
    const { activeItem } = this.state;
    return (
      <div style={{ backgroundColor: '#292e34', padding: '1rem' }}>
        <Nav onSelect={this.onSelect}>
          <NavList variant={NavVariants.horizontal}>
            <NavItem preventDefault to="#home" itemId={0} isActive={activeItem === 0}>
              Home
            </NavItem>
            <NavItem preventDefault to="#querylog" itemId={1} isActive={activeItem === 1}>
              Query Log
            </NavItem>
          </NavList>
        </Nav>
      </div>
    );
  }
}

export default NavHorizontalList;