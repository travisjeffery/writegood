import { Value } from 'slate'

const initialState = Value.fromJSON({
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

export default function editor(state = initialState, action) {
  switch (action.type) {
  case 'CHANGE_EDITOR':
    return action.value
  default:
    return state
  }
}
