import React, { Component } from 'react';
import ReactDOM from "react-dom";
import "@patternfly/patternfly/patternfly.css"
import "@patternfly/patternfly/components/Page/page.css"
import "@patternfly/react-core/dist/styles/base.css";
import { HorizontalPage } from './gudgeon'

ReactDOM.render(<HorizontalPage />, document.getElementById("root"));