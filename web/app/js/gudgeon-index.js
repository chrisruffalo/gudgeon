import React from 'react';
import ReactDOM from "react-dom";
import "@patternfly/patternfly/patternfly.min.css";
import '../css/gudgeon-app.css';
import "../css/gudgeon-overrides.css";
import { Gudgeon } from './gudgeon';

ReactDOM.render(<Gudgeon />, document.getElementById("root"));