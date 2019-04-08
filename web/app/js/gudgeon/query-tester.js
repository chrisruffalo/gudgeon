import React from 'react';
import Axios from 'axios';
import {
  ActionGroup,
  Card,
  CardBody,
  Button,
  Form,
  FormGroup,
  FormSelect,
  FormSelectOption,
  TextInput,
  GridItem,
  Grid
} from "@patternfly/react-core";
import { PrismAsyncLight as SyntaxHighlighter } from 'react-syntax-highlighter';
import { googlecode } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import gudgeonStyles from '../../css/gudgeon-app.css';
import { css } from '@patternfly/react-styles';


export class QueryTester extends React.Component {
  constructor(props) {
    super(props);
  };

  state = {
    qtype: "A",
    query: "",
    components: {},
    type: "consumer",
    target: "",
    loading: false
  };

  typeOptions = [
    { value: 'consumer', label: 'Consumer', disabled: false },
    { value: 'groups', label: 'Group', disabled: false },
    { value: 'resolvers', label: 'Resolver', disabled: false }
  ];

  qtypeOptions = [
    { value: 'A', label: 'A', disabled: false },
    { value: 'AAAA', label: 'AAAA', disabled: false },
    { value: 'PTR', label: 'PTR', disabled: false },
    { value: 'TXT', label: 'TXT', disabled: false },
    { value: 'ANY', label: 'ANY', disabled: false }
  ];

  onTypeChange = (value, event) => {
    this.setState({ type: value, target: this.state.components[value][0] });
  };

  onTargetChange = (value, event) => {
    this.setState({ target: value})
  };

  onQtypeChange = (value, event) => {
    this.setState({ qtype: value})
  };


  handleQueryChange = (value, event) => {
    this.setState( { query: value });
    event.preventDefault();
  };

  doQuery = (event) => {
    event.preventDefault();

    // don't query if null or empty
    if ( this.state.query === null || this.state.query.length < 1 ) {
      return;
    }

    // call into the query call after loading is established
    this.setState({loading: true}, this._queryCall);
  };

  _queryCall = () => {
    let options = {
      domain: this.state.query,
      qtype: this.state.qtype
    };
    options[this.state.type] = this.state.target;

    Axios
        .get("/api/test/query", { params: options } )
        .then(response => {
          this.setState({ response: response.data, loading: false });
        })
  };

  componentDidMount() {
    // create after-state set for loading callback
    let callback = () => {
      Axios
          .get("/api/test/components" )
          .then(response => {
            this.setState({ loading: false, components: response.data, target: response.data[this.state.type][0] });
          })
    };

    // set state for loading and callback
    this.setState({ loading: false}, callback);
  }

  componentWillUnmount() {

  }

  render() {
    let { components, response } = this.state;

    // setup secondary option
    let secondary = null;
    if ( components != null && components[this.state.type] != null ) {
      secondary = (
        <FormGroup
          label="Test Target"
          fieldId="query-target"
          helperText="Target Config for Query Test"
        >
          <FormSelect
            value={this.state.target}
            onChange={this.onTargetChange}
            id="query-target"
            name="query-target"
            isDisabled={ this.state.loading }
          >
            {components[this.state.type].map((option, index) => (
              <FormSelectOption isDisabled={false} key={index} value={option} label={option} />
            ))}
          </FormSelect>
        </FormGroup>
      );
    }

    // set up text box and form buttons
    let queryInput = null;
    let formButtons = null;
    let queryTypeSelect = null;

    if ( secondary !== null && this.state.target !== null ) {
      queryTypeSelect = (
        <FormGroup
            label="Query Type"
            fieldId="query-qtype"
            helperText="Type of Query to Make"
        >
          <FormSelect
                      value={this.state.qtype}
                      onChange={this.onQtypeChange}
                      id="query-qtype"
                      isDisabled={ this.state.loading }
          >
            {this.qtypeOptions.map((option, index) => (
                <FormSelectOption isDisabled={false} key={index} value={option.value} label={option.label} />
            ))}
          </FormSelect>
        </FormGroup>
      );

      queryInput = (
        <FormGroup
            label="Query String"
            fieldId="query-string"
            helperText="Domain Name to Query"
        >
          <TextInput
            value={this.state.query}
            onChange={this.handleQueryChange}
            id="query-string"
            name="query-string"
            isDisabled={ this.state.loading }
          />
        </FormGroup>
      );

      formButtons = (
        <ActionGroup>
          <Button isDisabled={this.state.loading || this.state.query == null || this.state.query.length < 1} onClick={ this.doQuery } variant="primary">Query</Button>
        </ActionGroup>
      );
    }

    let output = null;
    if ( response != null && response.result != null ) {
      let ruleMatch = "NONE";
      if ( response.result.Match === 1 ) {
        ruleMatch = "BLOCKED";
      } else if ( response.result.Match === 2 ) {
        ruleMatch = "ALLOWED";
      }

      output = (
        <GridItem lg={12} md={12} sm={12}>
          <Card>
            <CardBody>
              <table>
                <tbody>
                  <tr><td><strong>Cached</strong></td><td className={css(gudgeonStyles.queryLog)}>{ response.result.Cached ? "True" : "False" }</td></tr>
                  <tr><td><strong>Match Type</strong></td><td className={css(gudgeonStyles.queryLog)}>{ ruleMatch }</td></tr>
                  { ruleMatch === "BLOCKED" || ruleMatch === "ALLOWED" ? (<tr><td><strong>Match List</strong></td><td className={css(gudgeonStyles.queryLog)}>{ response.result.MatchList.Name }</td></tr>) : null }
                  { ruleMatch === "BLOCKED" || ruleMatch === "ALLOWED" ? (<tr><td><strong>Match Rule</strong></td><td className={css(gudgeonStyles.queryLog)}>{ response.result.MatchRule }</td></tr>) : null }
                </tbody>
              </table>
              { response.text != null && response.text.length > 0 ? <SyntaxHighlighter language='dns' style={googlecode}>{response.text}</SyntaxHighlighter> : null }
            </CardBody>
          </Card>
        </GridItem>
      );
    }

    return (
        <Grid gutter="sm">
          <GridItem lg={12} md={12} sm={12}>
            <Card>
              <CardBody>
                <Form isHorizontal onSubmit={ this.doQuery }>
                  <FormGroup
                      label="Test Type"
                      fieldId="query-test-type"
                      helperText="Type of Test to Run"
                  >
                    <FormSelect
                        value={this.state.type}
                        onChange={this.onTypeChange}
                        id="query-test-type"
                        name="query-test-type"
                        isDisabled={ this.state.loading }
                    >
                      {this.typeOptions.map((option, index) => (
                          <FormSelectOption isDisabled={option.disabled} key={index} value={option.value} label={option.label} />
                      ))}
                    </FormSelect>
                  </FormGroup>
                  { secondary }
                  { queryTypeSelect }
                  { queryInput }
                  { formButtons }
                </Form>
              </CardBody>
            </Card>
          </GridItem>
          { output }
        </Grid>
    );
  }
}