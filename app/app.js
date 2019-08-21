'use strict'

import React from 'react'
import ReactDOM from 'react-dom'
import { Editor } from 'slate-react'
import { Value } from 'slate'
import PasteLinkify from 'slate-paste-linkify'
import NoEmpty from 'slate-no-empty'
import Lists from '@convertkit/slate-lists'

// Create our initial value...
const initialValue = Value.fromJSON({
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

const plugins = [
  PasteLinkify(),
  NoEmpty('paragraph'),
  Lists({
    blocks: {
      ordered_list: "ordered-list",
      unordered_list: "unordered-list",
      list_item: "list-item",
    },
    classNames: {
      ordered_list: "ordered-list",
      unordered_list: "unordered-list",
      list_item: "list-item"
    }
  }),
]

class App extends React.Component {
  state = {
    value: initialValue,
  }

  onChange = ({ value }) => {
    this.setState({ value })
  }

  render() {
    return <Editor
             value={this.state.value}
             plugins={plugins}
             onChange={this.onChange}
           />
  }
}

ReactDOM.render(<App />, document.getElementById('app'))
