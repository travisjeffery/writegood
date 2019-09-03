'use strict'

import ApolloClient from 'apollo-boost'
import { gql } from "apollo-boost"
import Turbolinks from 'turbolinks'

Turbolinks.start()

const client = new ApolloClient({
  uri: 'http://localhost:8080/graphql',
});

client.
  query({query: gql`{user(id:1){id email}}`})
  .then(result => console.log(result.data));
