'use strict'

import ApolloClient from 'apollo-boost'
import { gql } from "apollo-boost"
import React from 'react'
import { render } from 'react-dom'
import { createStore } from 'redux'
import { Provider } from 'react-redux'
import App from './components/App'
import rootReducer from './reducers'

const client = new ApolloClient({
  uri: 'http://localhost:8080/graphql',
});

client.
  query({query: gql`{user(id:1){id email}}`})
  .then(result => console.log(result.data));

const store = createStore(
  rootReducer,
  window.__REDUX_DEVTOOLS_EXTENSION__ && window.__REDUX_DEVTOOLS_EXTENSION__()p
)

render(<Provider store={store}>
         <App />
       </Provider>,
       document.getElementById("app")
      )
