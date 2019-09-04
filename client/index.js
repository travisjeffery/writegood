'use strict'

import ApolloClient from 'apollo-boost'
import { gql } from "apollo-boost"
import React from 'react'
import { render } from 'react-dom'
import { createStore } from 'redux'
import { Provider } from 'react-redux'
import App from './components/App'
import rootReducer from './reducers'
import { Value } from 'slate'

const client = new ApolloClient({
  uri: 'http://localhost:8080/graphql',
});

client.
  query({query: gql`{user(id:1){id email}}`})
  .then(result => console.log(result.data));

const editor = Value.fromJSON({
  document: {
    nodes: [
      {
        object: 'block',
        type: 'paragraph',
        nodes: [
          {
            object: 'text',
            text: 'A line of text in a paragraph.',
          },
        ],
      },
    ],
  },
})

const documents = [{id: 1, text: "Hello world"}]

const store = createStore(rootReducer, {
  editor,
  documents,
})

render(<Provider store={store}>
         <App />
       </Provider>,
       document.getElementById("app")
      )
