'use strict'

import ApolloClient from 'apollo-boost'
import { gql } from "apollo-boost"
import React from 'react'
import { render } from 'react-dom'
import { createStore } from 'redux'
import { Provider } from 'react-redux'
import App from './components/App'
import rootReducer from './reducers'
import '@shopify/polaris/styles.css';
import { AppProvider } from '@shopify/polaris'
import { Frame, Page, Card, Layout } from '@shopify/polaris'
import TopBar from './components/TopBar'


const theme = {
  colors: {
    topBar: {
      background: '#fcda05',
    },
  },
  logo: {
    width: 124,
    topBarSource:
    '/static/logo_transparent.png',
    accessibilityLabel: 'WriteGood',
  },
};

const client = new ApolloClient({
  uri: 'http://localhost:8080/graphql',
});

client.
  query({query: gql`{user(id:1){id email}}`})
  .then(result => console.log(result.data));

const store = createStore(
  rootReducer,
  window.__REDUX_DEVTOOLS_EXTENSION__ && window.__REDUX_DEVTOOLS_EXTENSION__()
)

render(
  <AppProvider theme={theme}>
    <Provider store={store}>
      <Frame topBar={<TopBar />}>
        <App />
      </Frame>
    </Provider>
  </AppProvider>,
  document.getElementById("app")
)
