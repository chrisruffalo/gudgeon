import "@babel/polyfill";
import React, { Component } from 'react';
import ReactDOM from "react-dom";
import "material-icons/iconfont/material-icons.css";
import "@patternfly/react-core/dist/styles/base.css";
import { Gudgeon } from './gudgeon';
import "../css/gudgeon-overrides.css";

ReactDOM.render(<Gudgeon />, document.getElementById("root"));