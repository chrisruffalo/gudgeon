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
import SyntaxHighlighter from 'react-syntax-highlighter';
import { googlecode } from 'react-syntax-highlighter/dist/esm/styles/hljs';

export class QueryTester extends React.Component {
  constructor(props) {
    super(props);
  };

  state = {
    qtype: "A",
    query: "",
    components: {},
    type: "consumer",
    target: ""
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

    let options = {
      domain: this.state.query,
      qtype: this.state.qtype
    };
    options[this.state.type] = this.state.target;

    Axios
        .get("/api/test/query", { params: options } )
        .then(response => {
          this.setState({ response: response.data });
        })
  };

  componentDidMount() {
    Axios
      .get("/api/test/components" )
      .then(response => {
        this.setState({ components: response.data, target: response.data[this.state.type][0] });
      })
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
          />
        </FormGroup>
      );

      formButtons = (
        <ActionGroup>
          <Button isDisabled={this.state.query == null || this.state.query.length < 1} onClick={ this.doQuery } variant="primary">Query</Button>
        </ActionGroup>
      );
    }

    let output = null;
    if ( response != null && response.text != null ) {
      output = (
        <GridItem lg={12} md={12} sm={12}>
          <Card>
            <CardBody>
              <SyntaxHighlighter language='text' style={googlecode}>{response.text}</SyntaxHighlighter>
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